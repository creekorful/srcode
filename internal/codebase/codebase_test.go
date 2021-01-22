package codebase

import (
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

func TestCodebase_Projects(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedProjects := map[string]manifest.Project{
		"test/15": {Remote: "test"},
	}

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	manProviderMock.EXPECT().
		Read("test-dir/.srcode/manifest.json").
		Return(manifest.Manifest{
			Projects: expectedProjects,
		}, nil)

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	codebase := &codebase{
		manProvider:  manProviderMock,
		repoProvider: repoProviderMock,
		rootPath:     "test-dir",
	}

	repoProviderMock.EXPECT().Open(filepath.Join("test-dir", "test", "15"))

	projects, err := codebase.Projects()
	if err != nil {
		t.FailNow()
	}

	entry, exist := projects["test/15"]
	if !exist {
		t.Fail()
	}
	if entry.Project.Remote != "test" {
		t.Fail()
	}
}

func TestCodebase_Add(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	type test struct {
		repoRemote string // the repo remote
		localPath  string // the local path that should be computed for the project
		argPath    string // the wanted / provided path for the project
		from       string // from where we are opening the codebase (i.e codebase local path)
	}

	tests := []test{
		{
			repoRemote: "git@github.com:creekorful/test.git",
			localPath:  "test",
			argPath:    "",
			from:       "",
		},
		{
			repoRemote: "git@github.com:creekorful/test.git",
			localPath:  "Test-folder/Another/test",
			argPath:    "Test-folder/Another/test",
			from:       "",
		},
		{
			repoRemote: "git@github.com:creekorful/test.git",
			localPath:  "test/b/c/d/test",
			argPath:    "",
			from:       "test/b/c/d",
		},
		{
			repoRemote: "git@github.com:creekorful/test.git",
			localPath:  "test/d/a/b/Test-folder/Another/test",
			argPath:    "Test-folder/Another/test",
			from:       "test/d/a/b",
		},
	}

	for _, test := range tests {
		codebase := &codebase{
			repoProvider: repoProviderMock,
			repo:         repoMock,
			manProvider:  manProviderMock,
			rootPath:     "/home/creekorful",
			localPath:    test.from,
		}

		currentRepoMock := repository_mock.NewMockRepository(mockCtrl)

		repoProviderMock.EXPECT().
			Clone(test.repoRemote, filepath.Join(codebase.rootPath, test.localPath)).
			Return(currentRepoMock, nil)

		manProviderMock.EXPECT().
			Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
			Return(manifest.Manifest{}, nil)

		manProviderMock.EXPECT().
			Write(filepath.Join(codebase.rootPath, metaDir, manifestFile),
				manifest.Manifest{
					Projects: map[string]manifest.Project{
						test.localPath: {
							Remote: test.repoRemote,
							Config: map[string]string{
								"user.name":  "Aloïs Micard",
								"user.email": "alois@micard.lu",
							},
						},
					},
				}).
			Return(nil)

		repoMock.EXPECT().
			CommitFiles(fmt.Sprintf("Add %s to %s", test.repoRemote, test.localPath), "manifest.json").
			Return(nil)

		currentRepoMock.EXPECT().
			SetConfig("user.name", "Aloïs Micard").
			Return(nil)
		currentRepoMock.EXPECT().
			SetConfig("user.email", "alois@micard.lu").
			Return(nil)

		project, err := codebase.Add(test.repoRemote, test.argPath, map[string]string{
			"user.name":  "Aloïs Micard",
			"user.email": "alois@micard.lu",
		})
		if err != nil {
			t.Fail()
		}

		if project.Remote != test.repoRemote {
			t.Fail()
		}
		if project.Config["user.name"] != "Aloïs Micard" {
			t.Fail()
		}
		if project.Config["user.email"] != "alois@micard.lu" {
			t.Fail()
		}
	}
}

func TestCodebase_Add_PathTaken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	codebase := &codebase{
		repoProvider: repoProviderMock,
		repo:         repoMock,
		manProvider:  manProviderMock,
		rootPath:     "/home/creekorful",
		localPath:    "",
	}

	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Times(2).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/test":       {Remote: ""},
				"inside-dir/test": {Remote: ""},
			},
		}, nil)

	if _, err := codebase.Add("git@github.com:test/test.git", "test/test", nil); !errors.Is(err, ErrPathTaken) {
		t.Fail()
	}

	codebase.localPath = "inside-dir"
	if _, err := codebase.Add("git@github.com:test/test.git", "", nil); !errors.Is(err, ErrPathTaken) {
		t.Fail()
	}
}

func TestCodebase_Sync(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	dir := t.TempDir()

	codebase := &codebase{
		repoProvider: repoProviderMock,
		repo:         repoMock,
		manProvider:  manProviderMock,
		rootPath:     dir,
	}

	// create mock directory
	if err := os.MkdirAll(filepath.Join(dir, "test", "a", "b"), 0750); err != nil {
		t.FailNow()
	}

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/a/b": {Remote: "test.git"},
				"test/c/d": {Remote: "test.git"},
			},
		}, nil)

	repoMock.EXPECT().Pull("origin", "master").Return(nil)
	repoMock.EXPECT().Push("origin", "master").Return(nil)

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test-12": {
					Remote: "test.git",
					Config: map[string]string{
						"user.name": "Aloïs Micard",
					},
				},
				"test/c/d": {
					Remote: "test.git",
					Config: map[string]string{
						"user.mail": "alois@micard.lu",
					},
				},
			},
		}, nil)

	// should clone missing projects
	repoProviderMock.EXPECT().
		Clone("test.git", filepath.Join(dir, "test-12")).
		Return(nil, nil)

	cRepoMock := repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().Open(filepath.Join(dir, "test-12")).Return(cRepoMock, nil)
	cRepoMock.EXPECT().SetConfig("user.name", "Aloïs Micard").Return(nil) // restore config

	cRepoMock = repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().Open(filepath.Join(dir, "test/c/d")).Return(cRepoMock, nil)
	cRepoMock.EXPECT().SetConfig("user.mail", "alois@micard.lu").Return(nil)

	wg := sync.WaitGroup{}

	added := map[string]manifest.Project{}
	addedChan := make(chan ProjectEntry)
	go func() {
		wg.Add(1)
		for entry := range addedChan {
			added[entry.Path] = entry.Project
		}

		wg.Done()
	}()

	deleted := map[string]manifest.Project{}
	deletedChan := make(chan ProjectEntry)
	go func() {
		wg.Add(1)
		for entry := range deletedChan {
			deleted[entry.Path] = entry.Project
		}

		wg.Done()
	}()

	if err := codebase.Sync(true, addedChan, deletedChan); err != nil {
		t.FailNow()
	}

	wg.Wait()

	if len(added) != 1 || added["test-12"].Remote != "test.git" {
		t.Fail()
	}
	if len(deleted) != 1 || deleted["test/a/b"].Remote != "test.git" {
		t.Fail()
	}

	// make sure we have deleted the directory
	if _, err := os.Stat(filepath.Join(dir, "test", "a", "b")); !os.IsNotExist(err) {
		t.Fail()
	}
}

func TestCodebase_Sync_NoChannel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	dir := t.TempDir()

	codebase := &codebase{
		repoProvider: repoProviderMock,
		repo:         repoMock,
		manProvider:  manProviderMock,
		rootPath:     dir,
	}

	// create mock directory
	if err := os.MkdirAll(filepath.Join(dir, "test", "a", "b"), 0750); err != nil {
		t.FailNow()
	}

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/a/b": {Remote: "test.git"},
				"test/c/d": {Remote: "test.git"},
			},
		}, nil)

	repoMock.EXPECT().Pull("origin", "master").Return(nil)
	repoMock.EXPECT().Push("origin", "master").Return(nil)

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test-12": {
					Remote: "test.git",
					Config: map[string]string{
						"user.name": "Aloïs Micard",
					},
				},
				"test/c/d": {
					Remote: "test.git",
					Config: map[string]string{
						"user.mail": "alois@micard.lu",
					},
				},
			},
		}, nil)

	// should clone missing projects
	repoProviderMock.EXPECT().
		Clone("test.git", filepath.Join(dir, "test-12")).
		Return(nil, nil)

	cRepoMock := repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().Open(filepath.Join(dir, "test-12")).Return(cRepoMock, nil)
	cRepoMock.EXPECT().SetConfig("user.name", "Aloïs Micard").Return(nil) // restore config

	cRepoMock = repository_mock.NewMockRepository(mockCtrl)
	repoProviderMock.EXPECT().Open(filepath.Join(dir, "test/c/d")).Return(cRepoMock, nil)
	cRepoMock.EXPECT().SetConfig("user.mail", "alois@micard.lu").Return(nil)

	if err := codebase.Sync(true, nil, nil); err != nil {
		t.FailNow()
	}
}

func TestCodebase_Run(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)

	codebase := &codebase{
		manProvider: manProviderMock,
		rootPath:    "test-dir",
		localPath:   "",
	}

	manProviderMock.EXPECT().
		Read(filepath.Join("test-dir", metaDir, manifestFile)).
		Times(5).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something": {
					Commands: map[string]string{
						"greet-local":    "echo Hello from local command",
						"greet-global":   "@greet",
						"invalid-global": "@invalid",
					},
				},
			},
			Commands: map[string]string{
				"greet": "echo Hello from global command",
			},
		}, nil)

	// Try to run command from a non-project directory
	if _, err := codebase.Run("greet-local"); !errors.Is(err, ErrNoProjectFound) {
		t.Fail()
	}

	// CD inside a project
	codebase.localPath = "test/something"

	// Try to run an non existing local command
	if _, err := codebase.Run("blah"); !errors.Is(err, ErrCommandNotFound) {
		t.Fail()
	}

	// Try to run an non existing global command
	if _, err := codebase.Run("invalid-global"); !errors.Is(err, ErrCommandNotFound) {
		t.Fail()
	}

	// Try to run a local command
	if res, err := codebase.Run("greet-local"); err != nil || res != "Hello from local command" {
		t.Fail()
	}

	// Try to run a global command
	if res, err := codebase.Run("greet-global"); err != nil || res != "Hello from global command" {
		t.Fail()
	}
}

func TestCodebase_BulkGIT(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	codebase := &codebase{
		manProvider:  manProviderMock,
		repoProvider: repoProviderMock,
		rootPath:     "/etc/code",
		localPath:    "a",
	}

	manProviderMock.EXPECT().
		Read(filepath.Join("/", "etc", "code", metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {},
				"test/something-2": {},
			},
		}, nil)

	for _, path := range []string{"test/something-1", "test/something-2"} {
		repoMock := repository_mock.NewMockRepository(mockCtrl)
		repoProviderMock.EXPECT().
			Open(filepath.Join("/", "etc", "code", path)).
			Return(repoMock, nil)

		repoMock.EXPECT().
			RawCmd([]string{"pull", "--rebase"}).
			Return("", nil)
	}

	if err := codebase.BulkGIT([]string{"pull", "--rebase"}, nil); err != nil {
		t.Fail()
	}
}

func TestCodebase_BulkGIT_WithOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	codebase := &codebase{
		manProvider:  manProviderMock,
		repoProvider: repoProviderMock,
		rootPath:     "/etc/code",
		localPath:    "a",
	}

	manProviderMock.EXPECT().
		Read(filepath.Join("/", "etc", "code", metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {},
				"test/something-2": {},
			},
		}, nil)

	wg := sync.WaitGroup{}

	sb := strings.Builder{}
	ch := make(chan string)
	go func() {
		wg.Add(1)
		for res := range ch {
			sb.WriteString(res)
			sb.WriteString("\n")
		}
		wg.Done()
	}()

	for _, path := range []string{"test/something-1", "test/something-2"} {
		repoMock := repository_mock.NewMockRepository(mockCtrl)
		repoProviderMock.EXPECT().
			Open(filepath.Join("/", "etc", "code", path)).
			Return(repoMock, nil)

		repoMock.EXPECT().
			RawCmd([]string{"pull", "--rebase"}).
			Return(fmt.Sprintf("out: %s", path), nil)
	}

	if err := codebase.BulkGIT([]string{"pull", "--rebase"}, ch); err != nil {
		t.Fail()
	}

	wg.Wait()

	val := sb.String()
	if !strings.Contains(val, "out: test/something-1") {
		t.Fail()
	}
	if !strings.Contains(val, "out: test/something-2") {
		t.Fail()
	}
}

func TestCodebase_SetCommand(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	codebase := &codebase{
		manProvider: manProviderMock,
		repo:        repoMock,
		rootPath:    "test-dir",
	}

	manProviderMock.EXPECT().
		Read(filepath.Join("test-dir", metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something": {},
			},
		}, nil)

	// from should should only works with global = true
	if err := codebase.SetCommand("test", "test", false); !errors.Is(err, ErrNoProjectFound) {
		t.Fail()
	}

	manProviderMock.EXPECT().
		Read(filepath.Join("test-dir", metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something": {},
			},
		}, nil)

	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile),
			manifest.Manifest{
				Projects: map[string]manifest.Project{
					"test/something": {},
				},
				Commands: map[string]string{
					"test": "test",
				},
			}).
		Return(nil)

	repoMock.EXPECT().CommitFiles("Add command `test`: `test`", "manifest.json")

	// should works
	if err := codebase.SetCommand("test", "test", true); err != nil {
		t.Fail()
	}

	// we are inside project
	codebase.localPath = "test/something"

	manProviderMock.EXPECT().
		Read(filepath.Join("test-dir", metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something": {},
			},
		}, nil)

	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile),
			manifest.Manifest{
				Projects: map[string]manifest.Project{
					"test/something": {
						Commands: map[string]string{
							"test": "test",
						},
					},
				},
			}).
		Return(nil)

	repoMock.EXPECT().CommitFiles("Add command `test`: `test`", "manifest.json")

	if err := codebase.SetCommand("test", "test", false); err != nil {
		t.Fail()
	}

	manProviderMock.EXPECT().
		Read(filepath.Join("test-dir", metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something": {},
			},
		}, nil)

	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile),
			manifest.Manifest{
				Projects: map[string]manifest.Project{
					"test/something": {},
				},
				Commands: map[string]string{
					"test": "test",
				},
			}).
		Return(nil)

	repoMock.EXPECT().CommitFiles("Add command `test`: `test`", "manifest.json")

	if err := codebase.SetCommand("test", "test", true); err != nil {
		t.Fail()
	}
}

func TestCodebase_MoveProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	path := t.TempDir()

	codebase := &codebase{
		manProvider: manProviderMock,
		repo:        repoMock,
		rootPath:    path,
	}

	// Create dummy directories & files to simulate projects
	if err := os.MkdirAll(filepath.Join(path, "test", "something-1"), 0750); err != nil {
		t.FailNow()
	}
	if err := ioutil.WriteFile(filepath.Join(path, "test", "something-1", "README.md"),
		[]byte("Hello from something-1"), 0640); err != nil {
		t.FailNow()
	}
	if err := os.MkdirAll(filepath.Join(path, "test", "something-2"), 0750); err != nil {
		t.FailNow()
	}
	if err := ioutil.WriteFile(filepath.Join(path, "test", "something-2", "README.md"),
		[]byte("Hello from something-2"), 0640); err != nil {
		t.FailNow()
	}

	// Move src doesn't exist full path
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Times(6).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {Remote: "test-1.git"},
				"test/something-2": {Remote: "test-2.git"},
			},
		}, nil)

	if err := codebase.MoveProject("test-1/something", "test-2/something"); !errors.Is(err, ErrNoProjectFound) {
		t.Errorf("wrong error (got: %s, want: %s)", err, ErrNoProjectFound)
	}

	// Move src doesn't exist relative path
	codebase.localPath = "test-1"
	if err := codebase.MoveProject("something", "test-2/something"); !errors.Is(err, ErrNoProjectFound) {
		t.Errorf("wrong error (got: %s, want: %s)", err, ErrNoProjectFound)
	}

	// Move dest exist full path
	codebase.localPath = ""
	if err := codebase.MoveProject("test/something-1", "test/something-2"); !errors.Is(err, ErrPathTaken) {
		t.Errorf("wrong error (got: %s, want: %s)", err, ErrPathTaken)
	}

	// Move dest exist relative path
	codebase.localPath = "test"
	if err := codebase.MoveProject("something-1", "something-2"); !errors.Is(err, ErrPathTaken) {
		t.Errorf("wrong error (got: %s, want: %s)", err, ErrPathTaken)
	}

	// Move works full path
	codebase.localPath = ""

	repoMock.EXPECT().CommitFiles("Moved test-1.git from test/something-1 to test/something", "manifest.json")
	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something":   {Remote: "test-1.git"},
				"test/something-2": {Remote: "test-2.git"},
			},
		})

	if err := codebase.MoveProject("test/something-1", "test/something"); err != nil {
		t.Errorf("MoveProject() has failed: %s", err)
	}

	// Make sure we have really moved the files
	if _, err := os.Stat(filepath.Join(path, "test", "something-1")); !os.IsNotExist(err) {
		t.Errorf("We haven't moved %s", filepath.Join(path, "test", "something-1"))
	}
	if _, err := os.Stat(filepath.Join(path, "test", "something")); err != nil {
		t.Errorf("We haven't moved %s", filepath.Join(path, "test", "something-1"))
	}

	b, err := ioutil.ReadFile(filepath.Join(path, "test", "something", "README.md"))
	if err != nil {
		t.FailNow()
	}
	if string(b) != "Hello from something-1" {
		t.Fail()
	}

	// Move works relative path
	codebase.localPath = "test"

	repoMock.EXPECT().CommitFiles("Moved test-2.git from test/something-2 to test/something-1", "manifest.json")
	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something":   {Remote: "test-1.git"},
				"test/something-1": {Remote: "test-2.git"},
			},
		})

	if err := codebase.MoveProject("something-2", "something-1"); err != nil {
		t.Errorf("MoveProject() has failed: %s", err)
	}

	// Make sure we have really moved the files
	if _, err := os.Stat(filepath.Join(path, "test", "something-2")); !os.IsNotExist(err) {
		t.Errorf("We haven't moved %s", filepath.Join(path, "test", "something-2"))
	}
	if _, err := os.Stat(filepath.Join(path, "test", "something-1")); err != nil {
		t.Errorf("We haven't moved %s", filepath.Join(path, "test", "something-2"))
	}

	b, err = ioutil.ReadFile(filepath.Join(path, "test", "something-1", "README.md"))
	if err != nil {
		t.FailNow()
	}
	if string(b) != "Hello from something-2" {
		t.Fail()
	}
}
