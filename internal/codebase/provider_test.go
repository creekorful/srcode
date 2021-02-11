package codebase

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/manifest_mock"
	"github.com/creekorful/srcode/internal/repository_mock"
	"github.com/golang/mock/gomock"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestProvider_Init(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	provider := provider{repoProvider: repoProviderMock}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Repository provider fails
	repoProviderMock.EXPECT().Init(filepath.Join(targetDir, metaDir)).Return(nil, errors.New("test error"))
	if _, err := provider.Init(targetDir, "", false); err == nil {
		t.Error(err)
	}

	// Delete the directory to cleanup the state
	if err := os.RemoveAll(filepath.Join(targetDir, metaDir)); err != nil {
		t.FailNow()
	}

	// should create the initial commit
	repoMock.EXPECT().CommitFiles("Initial commit", "manifest.json", "README.md").Return(nil)
	// should add remote
	repoMock.EXPECT().AddRemote("origin", "git@github.com:creekorful/test.git").Return(nil)

	repoProviderMock.EXPECT().Init(filepath.Join(targetDir, metaDir)).Return(repoMock, nil)
	val, err := provider.Init(targetDir, "git@github.com:creekorful/test.git", false)
	if err != nil {
		t.Fatal(err)
	}

	// should create manifest file
	if _, err := os.Stat(filepath.Join(targetDir, metaDir, manifestFile)); err != nil {
		t.Error(err)
	}

	// should create README.md
	if _, err := os.Stat(filepath.Join(targetDir, metaDir, "README.md")); err != nil {
		t.Error(err)
	}

	if val.(*codebase).rootPath != targetDir {
		t.Fail()
	}
	if val.(*codebase).localPath != "" {
		t.Fail()
	}
	if val.LocalPath() != "" {
		t.Fail()
	}
}

func TestProvider_Init_WithImport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	provider := provider{repoProvider: repoProviderMock}

	// Simulate existing project folder
	targetDir := filepath.Join(t.TempDir(), "test-directory")
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "Contributing", "sudo"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "php"), 0750); err != nil {
		t.Fatal(err)
	}

	// simulate opening of these projects
	repoProviderMock.EXPECT().Exists(targetDir).Return(false)
	repoProviderMock.EXPECT().Exists(filepath.Join(targetDir, "Contributing")).Return(false)

	sudoRepoMock := repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().Exists(filepath.Join(targetDir, "Contributing", "sudo")).Return(true)
	repoProviderMock.EXPECT().
		Open(filepath.Join(targetDir, "Contributing", "sudo")).
		Return(sudoRepoMock, nil)
	sudoRepoMock.EXPECT().Remote("origin").Return("git@github.com:Debian/sudo.git", nil)

	phpRepoMock := repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().Exists(filepath.Join(targetDir, "php")).Return(true)
	repoProviderMock.EXPECT().
		Open(filepath.Join(targetDir, "php")).
		Return(phpRepoMock, nil)
	phpRepoMock.EXPECT().Remote("origin").Return("git@github.com:old-stuff/php.git", nil)

	// should create the initial commit
	repoMock.EXPECT().CommitFiles("Initial commit", "manifest.json", "README.md").Return(nil)
	// should add remote
	repoMock.EXPECT().AddRemote("origin", "git@github.com:creekorful/test.git").Return(nil)

	repoProviderMock.EXPECT().Init(filepath.Join(targetDir, metaDir)).Return(repoMock, nil)
	val, err := provider.Init(targetDir, "git@github.com:creekorful/test.git", true)
	if err != nil {
		t.Fatal(err)
	}

	// should create manifest file
	if _, err := os.Stat(filepath.Join(targetDir, metaDir, manifestFile)); err != nil {
		t.Error(err)
	}

	// decode manifest file to make sure projects have been imported
	var man manifest.Manifest
	f, err := os.Open(filepath.Join(targetDir, metaDir, manifestFile))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&man); err != nil {
		t.Fatal(err)
	}

	if len(man.Projects) != 2 {
		t.Fatal(err)
	}

	if project, exist := man.Projects["Contributing/sudo"]; !exist {
		t.Errorf("missing Contributing/sudo project")
	} else {
		if project.Remote != "git@github.com:Debian/sudo.git" {
			t.Errorf("wrong remote: %s", project.Remote)
		}
	}

	if project, exist := man.Projects["php"]; !exist {
		t.Errorf("missing php project")
	} else {
		if project.Remote != "git@github.com:old-stuff/php.git" {
			t.Errorf("wrong remote: %s", project.Remote)
		}
	}

	// should create README.md
	if _, err := os.Stat(filepath.Join(targetDir, metaDir, "README.md")); err != nil {
		t.Error(err)
	}

	if val.(*codebase).rootPath != targetDir {
		t.Fail()
	}
	if val.(*codebase).localPath != "" {
		t.Fail()
	}
	if val.LocalPath() != "" {
		t.Fail()
	}
}

func TestProvider_New_CodebaseExist(t *testing.T) {
	provider := provider{}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Simulate an existing codebase
	if err := os.MkdirAll(filepath.Join(targetDir, metaDir), 0770); err != nil {
		t.FailNow()
	}

	if _, err := provider.Init(targetDir, "", false); !errors.Is(err, ErrCodebaseAlreadyExist) {
		t.Errorf("wrong error (got: %s, want: %s)", err, ErrCodebaseAlreadyExist)
	}
}

func TestProvider_Open(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	provider := provider{repoProvider: repoProviderMock}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Codebase does not exist (yet)
	if _, err := provider.Open(targetDir); !errors.Is(err, ErrCodebaseNotExist) {
		t.Fail()
	}

	// Simulate an existing codebase
	if err := os.MkdirAll(filepath.Join(targetDir, metaDir), 0770); err != nil {
		t.FailNow()
	}

	// Repository provider fails
	repoProviderMock.EXPECT().Open(filepath.Join(targetDir, metaDir)).Return(nil, errors.New("test error"))
	if _, err := provider.Open(targetDir); err == nil {
		t.Fail()
	}

	repoProviderMock.EXPECT().Open(filepath.Join(targetDir, metaDir)).Return(nil, nil)
	val, err := provider.Open(targetDir)
	if err != nil {
		t.Fatal(err)
	}

	if val.(*codebase).rootPath != targetDir {
		t.Fail()
	}
	if val.(*codebase).localPath != "" {
		t.Fail()
	}
	if val.LocalPath() != "" {
		t.Fail()
	}
}

func TestProvider_Open_InSubDirectory(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	provider := provider{repoProvider: repoProviderMock}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Simulate an existing codebase
	if err := os.MkdirAll(filepath.Join(targetDir, metaDir), 0770); err != nil {
		t.FailNow()
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "test", "a", "b"), 0770); err != nil {
		t.FailNow()
	}

	// Repository provider fails
	repoProviderMock.EXPECT().Open(filepath.Join(targetDir, metaDir)).Return(nil, nil)
	val, err := provider.Open(filepath.Join(targetDir, "test", "a", "b")) // Open from inside the codebase
	if err != nil {
		t.Fatal(err)
	}

	if val.(*codebase).rootPath != targetDir {
		t.Fail()
	}
	if val.(*codebase).localPath != filepath.Join("test", "a", "b") {
		t.Fail()
	}
	if val.LocalPath() != filepath.Join("test", "a", "b") {
		t.Fail()
	}
}

func TestProvider_Clone_CodebaseExist(t *testing.T) {
	provider := provider{}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Simulate an existing codebase
	if err := os.MkdirAll(filepath.Join(targetDir, metaDir), 0770); err != nil {
		t.FailNow()
	}

	if _, err := provider.Clone("something", targetDir, nil); !errors.Is(err, ErrCodebaseAlreadyExist) {
		t.Fail()
	}
}

func TestProvider_Clone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	manifestProviderMock := manifest_mock.NewMockProvider(mockCtrl)

	provider := provider{repoProvider: repoProviderMock, manifestProvider: manifestProviderMock}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	wg := sync.WaitGroup{}

	buffer := ""
	ch := make(chan ProjectEntry)
	go func() {
		wg.Add(1)
		for entry := range ch {
			buffer = fmt.Sprintf("%s\n%s %s", buffer, entry.Path, entry.Project.Remote)
		}
		wg.Done()
	}()

	// Cloning has fail
	repoProviderMock.EXPECT().
		Clone("test-remote", filepath.Join(targetDir, metaDir)).
		Return(nil, errors.New("test error"))
	if _, err := provider.Clone("test-remote", targetDir, nil); err == nil {
		t.Fail()
	}

	repoProviderMock.EXPECT().
		Clone("test-remote", filepath.Join(targetDir, metaDir)).
		Return(nil, nil)

	// simulate codebase structure
	if err := os.MkdirAll(filepath.Join(targetDir, "test", "12", ".git", "hooks"), 0750); err != nil {
		t.Fatal()
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "test-another", ".git", "hooks"), 0750); err != nil {
		t.Fatal()
	}

	// Simulate dummy manifest
	manifestProviderMock.EXPECT().
		Read(filepath.Join(targetDir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/12": {
					Remote: "https://example.org/test.git",
					Config: map[string]string{"user.name": "Aloïs Micard"},
					Scripts: map[string][]string{
						"lint-12": {"go lint"},
					},
					Hook: "lint-12",
				},
				"test-another": {
					Remote: "git@example.org:example/test.git",
					Config: map[string]string{"user.email": "alois@micard.lu"},
					Scripts: map[string][]string{
						"lint-global": {"@global-lint"},
					},
					Hook: "lint-global",
				},
			},
			Scripts: map[string][]string{
				"global-lint": {"golint -w"},
			}}, nil)

	// We should clone the projects & configure them
	repoMock := repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().
		Clone("https://example.org/test.git", filepath.Join(targetDir, "test", "12"))
	repoProviderMock.EXPECT().
		Open(filepath.Join(targetDir, "test", "12")).Return(repoMock, nil)
	repoMock.EXPECT().SetConfig("user.name", "Aloïs Micard").Return(nil)

	repoMock = repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().
		Clone("git@example.org:example/test.git", filepath.Join(targetDir, "test-another"))
	repoProviderMock.EXPECT().
		Open(filepath.Join(targetDir, "test-another")).Return(repoMock, nil)
	repoMock.EXPECT().SetConfig("user.email", "alois@micard.lu").Return(nil)

	val, err := provider.Clone("test-remote", targetDir, ch)
	if err != nil {
		t.Fail()
	}

	wg.Wait()

	if val.(*codebase).rootPath != targetDir {
		t.Fail()
	}
	if val.(*codebase).localPath != "" {
		t.Fail()
	}
	if val.LocalPath() != "" {
		t.Fail()
	}

	// validate cloning progress system
	if !strings.Contains(buffer, "test/12 https://example.org/test.git") {
		t.Fail()
	}
	if !strings.Contains(buffer, "test-another git@example.org:example/test.git") {
		t.Fail()
	}

	// make sure hook is copied
	b, err := ioutil.ReadFile(filepath.Join(targetDir, "test", "12", ".git", "hooks", "pre-push"))
	if err != nil {
		t.Fail()
	}
	if string(b) != "go lint" {
		t.Fatalf("got: %s want: go lint", string(b))
	}

	// make sure hook is copied
	b, err = ioutil.ReadFile(filepath.Join(targetDir, "test-another", ".git", "hooks", "pre-push"))
	if err != nil {
		t.Fail()
	}
	if string(b) != "golint -w" {
		t.Fatal()
	}
}
