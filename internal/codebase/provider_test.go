package codebase

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/manifest_mock"
	"github.com/creekorful/srcode/internal/repository_mock"
	"github.com/golang/mock/gomock"
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
	if _, err := provider.Init(targetDir, ""); err == nil {
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
	val, err := provider.Init(targetDir, "git@github.com:creekorful/test.git")
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

func TestProvider_New_CodebaseExist(t *testing.T) {
	provider := provider{}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Simulate an existing codebase
	if err := os.MkdirAll(filepath.Join(targetDir, metaDir), 0770); err != nil {
		t.FailNow()
	}

	if _, err := provider.Init(targetDir, ""); !errors.Is(err, ErrCodebaseAlreadyExist) {
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

	// Simulate dummy manifest
	manifestProviderMock.EXPECT().
		Read(filepath.Join(targetDir, metaDir, manifestFile)).
		Return(manifest.Manifest{Projects: map[string]manifest.Project{
			"test/12": {
				Remote: "https://example.org/test.git",
				Config: map[string]string{"user.name": "Aloïs Micard"},
			},
			"test-another": {
				Remote: "git@example.org:example/test.git",
				Config: map[string]string{"user.email": "alois@micard.lu"},
			},
		}}, nil)

	// We should clone the projects & configure them
	repoMock := repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().
		Clone("https://example.org/test.git", filepath.Join(targetDir, "test", "12")).Return(repoMock, nil)
	repoMock.EXPECT().SetConfig("user.name", "Aloïs Micard").Return(nil)

	repoMock = repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().
		Clone("git@example.org:example/test.git", filepath.Join(targetDir, "test-another")).Return(repoMock, nil)
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
}
