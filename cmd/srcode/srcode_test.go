package main

import (
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/creekorful/srcode/internal/codebase_mock"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/golang/mock/gomock"
	"os"
	"path/filepath"
	"testing"
)

// TODO check command output and test it

func TestInitCodebase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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
	codebaseProviderMock.EXPECT().Init(filepath.Join(cwd, "code"), "")
	if err := app.getCliApp().Run([]string{"srcode", "init", "code"}); err != nil {
		t.Fail()
	}

	// test init full path
	codebaseProviderMock.EXPECT().Init(filepath.Join("/", "etc", "code"), "git@github.com:test.git")
	if err := app.getCliApp().Run([]string{"srcode", "init", "--remote", "git@github.com:test.git", "/etc/code"}); err != nil {
		t.Fail()
	}
}

func TestCloneCodebase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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

	// test clone full path
	codebaseProviderMock.EXPECT().
		Clone("git@github.com:test.git", filepath.Join("/", "etc", "code"), gomock.Any()).
		Do(func(remote, path string, ch chan<- codebase.ProjectEntry) { close(ch) })
	if err := app.getCliApp().Run([]string{"srcode", "clone", "git@github.com:test.git", "/etc/code"}); err != nil {
		t.FailNow()
	}

	// test clone no path
	codebaseProviderMock.EXPECT().
		Clone("git@github.com:test.git", cwd, gomock.Any()).
		Do(func(remote, path string, ch chan<- codebase.ProjectEntry) { close(ch) })
	if err := app.getCliApp().Run([]string{"srcode", "clone", "git@github.com:test.git"}); err != nil {
		t.FailNow()
	}
}

func TestAddProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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

	// test with target path
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().
		Add("https://example.com/test.git", "Contributing/test", map[string]string{}).
		Return(manifest.Project{}, nil)

	if err := app.getCliApp().Run([]string{"srcode", "add", "https://example.com/test.git", "Contributing/test"}); err != nil {
		t.Fail()
	}

	// test with git configuration
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
}

func TestSyncCodebase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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
		Do(func(delete bool, ch1, ch2 chan<- codebase.ProjectEntry) { close(ch1); close(ch2) }).
		Return(nil)

	if err := app.getCliApp().Run([]string{"srcode", "sync"}); err != nil {
		t.Fail()
	}

	// test sync codebase with delete
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	// test sync no delete
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

	app := app{
		codebaseProvider: codebaseProviderMock,
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
}

func TestRunCmd(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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

	codebaseMock.EXPECT().Run("test")

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "run", "test"}); err != nil {
		t.Fail()
	}
}

func TestLsProjects(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)

	codebaseMock.EXPECT().Projects().
		Return(map[string]manifest.Project{
			"Contributing/test": {Remote: "https://example/test.git"},
			"Another/Stuff":     {Remote: "https://old.example/stuff.git"},
		}, nil)

	// test with no args should fails
	if err := app.getCliApp().Run([]string{"srcode", "ls"}); err != nil {
		t.Fail()
	}
}

func TestBulkGit(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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
		BulkGIT([]string{"pull", "--rebase", "--prune"}, gomock.Any()).
		Do(func(args []string, ch chan<- string) { close(ch) })

	if err := app.getCliApp().Run([]string{"srcode", "bulk-git", "pull", "--rebase", "--prune"}); err != nil {
		t.Fail()
	}
}

func TestSetCmd(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	// test with no enough args should fails
	if err := app.getCliApp().Run([]string{"srcode", "set-cmd"}); err != errWrongSetCmdUsage {
		t.Errorf("got %v want %v", err, errWrongSetCmdUsage)
	}
	if err := app.getCliApp().Run([]string{"srcode", "set-cmd", "test"}); err != errWrongSetCmdUsage {
		t.Errorf("got %v want %v", err, errWrongSetCmdUsage)
	}

	codebaseMock := codebase_mock.NewMockCodebase(mockCtrl)

	// test set local command
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().SetCommand("test", "@go-test", false)

	if err := app.getCliApp().Run([]string{"srcode", "set-cmd", "test", "@go-test"}); err != nil {
		t.Fail()
	}

	// test set global command
	codebaseProviderMock.EXPECT().Open(cwd).Return(codebaseMock, nil)
	codebaseMock.EXPECT().SetCommand("go-test", "go test -race -v ./...", true)

	if err := app.getCliApp().Run([]string{"srcode", "set-cmd", "--global", "go-test", "go", "test", "-race", "-v", "./..."}); err != nil {
		t.Fail()
	}
}

func TestMvProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	codebaseProviderMock := codebase_mock.NewMockProvider(mockCtrl)

	app := app{
		codebaseProvider: codebaseProviderMock,
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
