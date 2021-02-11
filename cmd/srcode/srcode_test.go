package main

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/creekorful/srcode/internal/codebase_mock"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository_mock"
	"github.com/creekorful/srcode/internal/str"
	"github.com/golang/mock/gomock"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCodebase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)
	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "init"}); err != errWrongInitUsage {
		t.Errorf("got %v want %v", err, errWrongInitUsage)
	}

	// test init relative path

	codebaseMock.EXPECT().Projects().Return(map[string]codebase.ProjectEntry{}, nil)

	codebaseProviderMock.EXPECT().Init(filepath.Join(cwd, "code"), "", false).
		Return(codebaseMock, nil)
	if err := app.getCliApp().Run([]string{"srcode", "init", "code"}); err != nil {
		t.Fail()
	}
	if b.String() != fmt.Sprintf("Successfully initialized new codebase at: %s\n", filepath.Join(cwd, "code")) {
		t.Fail()
	}

	// test init full path
	b.Reset()

	codebaseMock.EXPECT().Projects().Return(map[string]codebase.ProjectEntry{
		"Contributing/Test": {Project: manifest.Project{Remote: "test.git"}},
		"example":           {Project: manifest.Project{Remote: "example.git"}},
	}, nil)

	codebaseProviderMock.EXPECT().
		Init(filepath.Join("/", "etc", "code"), "git@github.com:test.git", true).
		Return(codebaseMock, nil)
	if err := app.getCliApp().Run([]string{"srcode", "init", "--import", "--remote", "git@github.com:test.git", "/etc/code"}); err != nil {
		t.Fail()
	}

	val := b.String()
	if !strings.Contains(val, fmt.Sprintf("Successfully initialized new codebase at: %s\n", filepath.Join("/", "etc", "code"))) {
		t.Fail()
	}
	if !strings.Contains(val, fmt.Sprintf("Imported test.git -> /Contributing/Test")) {
		t.Fail()
	}
	if !strings.Contains(val, fmt.Sprintf("Imported example.git -> /example")) {
		t.Fail()
	}
}

func TestCloneCodebase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &str.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "clone"}); err != errWrongCloneUsage {
		t.Errorf("got %v want %v", err, errWrongCloneUsage)
	}

	// test clone relative path
	codebaseProviderMock.EXPECT().
		Clone("git@github.com:test.git", filepath.Join(cwd, "code"), gomock.Any()).
		Do(func(remote, path string, ch chan<- codebase.ProjectEntry) { close(ch) })
	if err := app.getCliApp().Run([]string{"srcode", "clone", "git@github.com:test.git", "code"}); err != nil {
		t.FailNow()
	}

	if b.String() != fmt.Sprintf("Successfully cloned codebase from git@github.com:test.git to: %s\n", filepath.Join(cwd, "code")) {
		t.Fail()
	}

	// test clone full path
	b.Reset()
	codebaseProviderMock.EXPECT().
		Clone("git@github.com:test.git", filepath.Join("/", "etc", "code"), gomock.Any()).
		Do(func(remote, path string, ch chan<- codebase.ProjectEntry) { close(ch) })
	if err := app.getCliApp().Run([]string{"srcode", "clone", "git@github.com:test.git", "/etc/code"}); err != nil {
		t.FailNow()
	}

	if b.String() != fmt.Sprintf("Successfully cloned codebase from git@github.com:test.git to: %s\n", filepath.Join("/", "etc", "code")) {
		t.Fail()
	}

	// test clone no path
	b.Reset()
	codebaseProviderMock.EXPECT().
		Clone("git@github.com:test.git", cwd, gomock.Any()).
		Do(func(remote, path string, ch chan<- codebase.ProjectEntry) {
			ch <- codebase.ProjectEntry{
				Path:    "Contributing/Test",
				Project: manifest.Project{Remote: "test.git"},
			}
			close(ch)
		})
	if err := app.getCliApp().Run([]string{"srcode", "clone", "git@github.com:test.git"}); err != nil {
		t.FailNow()
	}

	val := b.String()
	if !strings.Contains(val, fmt.Sprintf("Successfully cloned codebase from git@github.com:test.git to: %s\n", cwd)) {
		t.Fail()
	}
	if !strings.Contains(val, "Cloned test.git -> /Contributing/Test") {
		t.Fail()
	}
}

func TestAddProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "add"}); err != errWrongAddProjectUsage {
		t.Errorf("got %v want %v", err, errWrongAddProjectUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)

	// test empty path
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().
		Add("https://example.com/test.git", "", map[string]string{}).
		Return(manifest.Project{}, nil)

	if err := app.getCliApp().Run([]string{"srcode", "add", "https://example.com/test.git"}); err != nil {
		t.Fail()
	}
	if b.String() != "Successfully added https://example.com/test.git to: /\n" {
		t.Fail()
	}

	// test with target path
	b.Reset()
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().
		Add("https://example.com/test.git", "Contributing/test", map[string]string{}).
		Return(manifest.Project{}, nil)

	if err := app.getCliApp().Run([]string{"srcode", "add", "https://example.com/test.git", "Contributing/test"}); err != nil {
		t.Fail()
	}
	if b.String() != "Successfully added https://example.com/test.git to: /Contributing/test\n" {
		t.Fail()
	}

	// test with git configuration
	b.Reset()
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().
		Add("https://example.com/test.git", "Contributing/test",
			map[string]string{"user.name": "Aloïs Micard", "user.email": "alois@micard.lu"}).
		Return(manifest.Project{}, nil)

	if err := app.getCliApp().Run([]string{"srcode", "add",
		"--git-config", "user.name=Aloïs Micard",
		"--git-config", "user.email=alois@micard.lu",
		"https://example.com/test.git", "Contributing/test"}); err != nil {
		t.Fail()
	}
	if b.String() != "Successfully added https://example.com/test.git to: /Contributing/test\n" {
		t.Fail()
	}
}

func TestSyncCodebase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &str.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	// test sync no delete
	codebaseMock.EXPECT().
		Sync(false, gomock.Any(), gomock.Any()).
		Do(func(delete bool, ch1, ch2 chan<- codebase.ProjectEntry) {
			ch1 <- codebase.ProjectEntry{
				Path:    "Test/12",
				Project: manifest.Project{Remote: "test-12.git"},
			}
			ch2 <- codebase.ProjectEntry{
				Path:    "Test/42",
				Project: manifest.Project{Remote: "test-42.git"},
			}
			close(ch1)
			close(ch2)
		}).
		Return(nil)

	if err := app.getCliApp().Run([]string{"srcode", "sync"}); err != nil {
		t.Fail()
	}

	val := b.String()

	if !strings.Contains(val, "Successfully synchronized codebase") {
		t.Fail()
	}
	if !strings.Contains(val, "[-] test-42.git -> Test/42") {
		t.Fail()
	}
	if !strings.Contains(val, "[+] test-12.git -> Test/12") {
		t.Fail()
	}

	// test sync codebase with delete
	b.Reset()
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().
		Sync(true, gomock.Any(), gomock.Any()).
		Do(func(delete bool, ch1, ch2 chan<- codebase.ProjectEntry) { close(ch1); close(ch2) }).
		Return(nil)

	if err := app.getCliApp().Run([]string{"srcode", "sync", "--delete-removed"}); err != nil {
		t.Fail()
	}
}

func TestPwd(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           &b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	codebaseMock.EXPECT().LocalPath().Return("Contributing")

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "pwd"}); err != nil {
		t.FailNow()
	}
	if b.String() != "/Contributing\n" {
		t.Fail()
	}
}

func TestRunScript(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "run"}); err != errWrongRunUsage {
		t.Errorf("got %v want %v", err, errWrongRunUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	codebaseMock.EXPECT().Run("test", []string{"."}, b).
		Do(func(command string, args []string, writer io.Writer) { _, _ = io.WriteString(writer, "test 42\n") }).
		Return(nil)

	if err := app.getCliApp().Run([]string{"srcode", "run", "test", "."}); err != nil {
		t.Fail()
	}
	if b.String() != "test 42\n" {
		t.Fail()
	}
}

func TestLsProjects(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	// No projects
	codebaseMock.EXPECT().Projects().Return(map[string]codebase.ProjectEntry{}, nil)

	if err := app.getCliApp().Run([]string{"srcode", "ls"}); err != nil {
		t.Fail()
	}

	val := b.String()
	if !strings.Contains(val, "No projects in codebase") {
		t.Fail()
	}
	if !strings.Contains(val, "srcode add git@github.com:darkspot-org/bathyscaphe.git Darkspot/bathyscaphe") {
		t.Fail()
	}

	b.Reset()

	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	repo1 := repository_mock.NewMockRepository(mockCtrl)
	repo1.EXPECT().Head().Return("develop", nil)
	repo1.EXPECT().IsDirty().Return(false, nil)

	repo2 := repository_mock.NewMockRepository(mockCtrl)
	repo2.EXPECT().Head().Return("main", nil)
	repo2.EXPECT().IsDirty().Return(true, nil)

	codebaseMock.EXPECT().Projects().
		Return(map[string]codebase.ProjectEntry{
			"Contributing/test": {
				Project:    manifest.Project{Remote: "https://example/test.git"},
				Repository: repo1,
			},
			"Another/Stuff": {
				Project:    manifest.Project{Remote: "https://old.example/stuff.git"},
				Repository: repo2,
			},
		}, nil)

	if err := app.getCliApp().Run([]string{"srcode", "ls"}); err != nil {
		t.Fail()
	}

	val = b.String()
	if !strings.Contains(val, "Contributing/test") {
		t.Fail()
	}
	if !strings.Contains(val, "https://example/test.git") {
		t.Fail()
	}
	if !strings.Contains(val, "develop") {
		t.Fail()
	}
	if !strings.Contains(val, "Another/Stuff") {
		t.Fail()
	}
	if !strings.Contains(val, "https://old.example/stuff.git") {
		t.Fail()
	}
	if !strings.Contains(val, "main") {
		t.Fail()
	}
	if !strings.Contains(val, "(*)") {
		t.Fail()
	}
}

func TestBulkGit(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "bulk-git"}); err != errWrongBulkGitUsage {
		t.Errorf("got %v want %v", err, errWrongBulkGitUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	codebaseMock.EXPECT().
		BulkGIT([]string{"pull", "--rebase", "--prune"}, b)

	if err := app.getCliApp().Run([]string{"srcode", "bulk-git", "pull", "--rebase", "--prune"}); err != nil {
		t.Fail()
	}
}

func TestScript(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)

	// test no project in current directory
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().Manifest().Return(manifest.Manifest{}, nil)
	codebaseMock.EXPECT().LocalPath().Return("test-12")

	// test script with no args should print existing scripts
	if err := app.getCliApp().Run([]string{"srcode", "script"}); err != nil {
		t.Fail()
	}

	if b.String() != "No scripts in codebase\n" {
		t.Fail()
	}

	// test no project in current directory
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().Manifest().Return(manifest.Manifest{}, nil)
	codebaseMock.EXPECT().LocalPath().Return("test-12")

	if err := app.getCliApp().Run([]string{"srcode", "script", "test", "@go-test"}); !errors.Is(err, manifest.ErrNoProjectFound) {
		t.Fail()
	}

	// test set local command
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().Manifest().Return(manifest.Manifest{Projects: map[string]manifest.Project{"test-12": {}}}, nil)
	codebaseMock.EXPECT().LocalPath().Return("test-12")
	codebaseMock.EXPECT().SetScript("test", []string{"@go-test"}, false)

	if err := app.getCliApp().Run([]string{"srcode", "script", "test", "@go-test"}); err != nil {
		t.Fail()
	}

	// test set global command
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().Manifest().Return(manifest.Manifest{Projects: map[string]manifest.Project{"test-42": {}}}, nil)
	codebaseMock.EXPECT().LocalPath().Return("")
	codebaseMock.EXPECT().SetScript("go-test", []string{"go test -race -v ./..."}, true)

	if err := app.getCliApp().Run([]string{"srcode", "script", "--global", "go-test", "go", "test", "-race", "-v", "./..."}); err != nil {
		t.Fail()
	}

	// test display commands
	b.Reset()

	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().Manifest().Return(manifest.Manifest{
		Projects: map[string]manifest.Project{"test-42": {Scripts: map[string][]string{"gen": {"@go-gen"}}}},
		Scripts:  map[string][]string{"go-generate": {"go generate -v ./..."}},
	}, nil)
	codebaseMock.EXPECT().LocalPath().Return("test-42")

	if err := app.getCliApp().Run([]string{"srcode", "script"}); err != nil {
		t.Fail()
	}

	val := b.String()
	if !strings.Contains(val, "available global scripts") {
		t.Fail()
	}
	if !strings.Contains(val, "go-generate") {
		t.Fail()
	}
	if !strings.Contains(val, "available local scripts") {
		t.Fail()
	}
	if !strings.Contains(val, "gen") {
		t.Fail()
	}
}

func TestMvProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no enough args should fails
	if err := app.getCliApp().Run([]string{"srcode", "mv"}); err != errWrongMvUsage {
		t.Errorf("got %v want %v", err, errWrongMvUsage)
	}
	if err := app.getCliApp().Run([]string{"srcode", "mv", "test"}); err != errWrongMvUsage {
		t.Errorf("got %v want %v", err, errWrongMvUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	codebaseMock.EXPECT().
		MoveProject("Perso/SuperStuff", "OldStuff/SuperStuff")

	if err := app.getCliApp().Run([]string{"srcode", "mv", "Perso/SuperStuff", "OldStuff/SuperStuff"}); err != nil {
		t.Fail()
	}
	if b.String() != "Successfully moved from Perso/SuperStuff to OldStuff/SuperStuff\n" {
		t.Fail()
	}
}

func TestRmProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no enough args should fails
	if err := app.getCliApp().Run([]string{"srcode", "rm"}); err != errWrongRmUsage {
		t.Errorf("got %v want %v", err, errWrongRmUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().RmProject("Contributing/Test", false)
	if err := app.getCliApp().Run([]string{"srcode", "rm", "Contributing/Test"}); err != nil {
		t.Fail()
	}

	b.Reset()
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().RmProject("Contributing/Test", true)
	if err := app.getCliApp().Run([]string{"srcode", "rm", "--delete", "Contributing/Test"}); err != nil {
		t.Fail()
	}

	if b.String() != "Successfully deleted Contributing/Test\n" {
		t.Fail()
	}
}

func TestHook(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	b := &strings.Builder{}

	app := app{
		codebaseProvider: codebaseProviderMock,
		writer:           b,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no enough args should fails
	if err := app.getCliApp().Run([]string{"srcode", "hook"}); err != errWrongHookUsage {
		t.Errorf("got %v want %v", err, errWrongHookUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().SetHook("lint").Return(nil)
	codebaseMock.EXPECT().LocalPath().Return("Contributing/Test")
	if err := app.getCliApp().Run([]string{"srcode", "hook", "lint"}); err != nil {
		t.Fail()
	}

	if b.String() != "Successfully applied hook `lint` to /Contributing/Test\n" {
		t.Fail()
	}
}

func TestParseGitConfig(t *testing.T) {
	config := parseGitConfig([]string{})
	if len(config) != 0 {
		t.Fail()
	}

	config = parseGitConfig([]string{"user.name=test", "user.email=something", "invalid"})
	if len(config) != 2 {
		t.Fail()
	}
	if config["user.name"] != "test" {
		t.Fail()
	}
	if config["user.email"] != "something" {
		t.Fail()
	}
}
