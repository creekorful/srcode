package codebase

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/manifest_mock"
	"github.com/creekorful/srcode/internal/repository_mock"
	"github.com/golang/mock/gomock"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
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
	if err := os.MkdirAll(filepath.Join(dir, "test-12", ".git", "hooks"), 0750); err != nil {
		t.Fatal()
	}
	if err := os.MkdirAll(filepath.Join(dir, "test", "c", "d", ".git", "hooks"), 0750); err != nil {
		t.Fatal()
	}

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/a/b": {Remote: "test.git"},
				"test/c/d": {Remote: "test.git"},
			},
		}, nil)

	repoMock.EXPECT().Pull("origin", "main").Return(nil)
	repoMock.EXPECT().Push("origin", "main").Return(nil)

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test-12": {
					Remote: "test.git",
					Config: map[string]string{
						"user.name": "Aloïs Micard",
					},
					Scripts: map[string][]string{
						"test-local": {"go test -v"},
					},
					Hook: "test-local",
				},
				"test/c/d": {
					Remote: "test.git",
					Config: map[string]string{
						"user.mail": "alois@micard.lu",
					},
					Scripts: map[string][]string{
						"test-global": {"@global-test"},
					},
					Hook: "test-global",
				},
			},
			Scripts: map[string][]string{
				"global-test": {"go test"},
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

	// make sure hook is copied
	b, err := ioutil.ReadFile(filepath.Join(dir, "test-12", ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Error(err)
	}
	if string(b) != "go test -v" {
		t.Fatalf("got: %s want: go lint", string(b))
	}

	// make sure hook is copied
	b, err = ioutil.ReadFile(filepath.Join(dir, "test", "c", "d", ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Error(err)
	}
	if string(b) != "go test" {
		t.Error(err)
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

	repoMock.EXPECT().Pull("origin", "main").Return(nil)
	repoMock.EXPECT().Push("origin", "main").Return(nil)

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

	b := &strings.Builder{}

	manProviderMock.EXPECT().
		Read(filepath.Join("test-dir", metaDir, manifestFile)).
		Times(6).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something": {
					Scripts: map[string][]string{
						"greet-local":    {"echo Hello from local script"},
						"greet-global":   {"@greet"},
						"invalid-global": {"@invalid"},
						"greet-custom":   {"@greet-custom"},
					},
				},
			},
			Scripts: map[string][]string{
				"greet":        {"echo Hello from global script"},
				"greet-custom": {"echo Hello $2 $1"},
			},
		}, nil)

	// Try to run script from a non-project directory
	if err := codebase.Run("greet-local", nil, b); !errors.Is(err, manifest.ErrNoProjectFound) {
		t.Fail()
	}

	// CD inside a project
	codebase.localPath = "test/something"

	// Try to run an non existing local script
	if err := codebase.Run("blah", nil, b); !errors.Is(err, manifest.ErrScriptNotFound) {
		t.Fail()
	}

	// Try to run an non existing global script
	if err := codebase.Run("invalid-global", nil, b); !errors.Is(err, manifest.ErrScriptNotFound) {
		t.Fail()
	}

	// Try to run a local script
	b.Reset()
	if err := codebase.Run("greet-local", nil, b); err != nil || b.String() != "Hello from local script\n" {
		t.Errorf("error: %v", err)
		t.Errorf("got: '%s' want: '%s'", b.String(), "Hello from local script")
	}

	// Try to run a global script
	b.Reset()
	if err := codebase.Run("greet-global", nil, b); err != nil || b.String() != "Hello from global script\n" {
		t.Errorf("error: %v", err)
		t.Errorf("got: '%s' want: '%s'", b.String(), "Hello from global script")
	}

	// Try to run a global custom script
	b.Reset()
	if err := codebase.Run("greet-custom", []string{"param1", "param2"}, b); err != nil || b.String() != "Hello param2 param1\n" {
		t.Errorf("error: %v", err)
		t.Errorf("got: '%s' want: '%s'", b.String(), "Hello param2 param1")
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

	sb := &strings.Builder{}
	for _, path := range []string{"test/something-1", "test/something-2"} {
		repoMock := repository_mock.NewMockRepository(mockCtrl)
		repoProviderMock.EXPECT().
			Open(filepath.Join("/", "etc", "code", path)).
			Return(repoMock, nil)

		repoMock.EXPECT().
			RawCmd([]string{"pull", "--rebase"}, sb).
			Return(nil)
	}

	if err := codebase.BulkGIT([]string{"pull", "--rebase"}, sb); err != nil {
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

	sb := &strings.Builder{}
	for _, path := range []string{"test/something-1", "test/something-2"} {
		repoMock := repository_mock.NewMockRepository(mockCtrl)
		repoProviderMock.EXPECT().
			Open(filepath.Join("/", "etc", "code", path)).
			Return(repoMock, nil)

		repoMock.EXPECT().
			RawCmd([]string{"pull", "--rebase"}, sb).
			Do(func(path string) func([]string, io.Writer) {
				return func(args []string, w io.Writer) {
					_, _ = io.WriteString(w, fmt.Sprintf("out: %s", path))
				}
			}(path)).Return(nil)
	}

	if err := codebase.BulkGIT([]string{"pull", "--rebase"}, sb); err != nil {
		t.Fail()
	}

	val := sb.String()

	if !strings.Contains(val, "out: test/something-1") {
		t.Fail()
	}
	if !strings.Contains(val, "out: test/something-2") {
		t.Fail()
	}
}

func TestCodebase_SetScript(t *testing.T) {
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
	if err := codebase.SetScript("test", []string{"test"}, false); !errors.Is(err, manifest.ErrNoProjectFound) {
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
				Scripts: map[string][]string{
					"test": {"test"},
				},
			}).
		Return(nil)

	repoMock.EXPECT().CommitFiles("Add global script `test`", "manifest.json")

	// should works
	if err := codebase.SetScript("test", []string{"test"}, true); err != nil {
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
						Scripts: map[string][]string{
							"test": {"test"},
						},
					},
				},
			}).
		Return(nil)

	repoMock.EXPECT().CommitFiles("Add script `test` to /test/something", "manifest.json")

	if err := codebase.SetScript("test", []string{"test"}, false); err != nil {
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
				Scripts: map[string][]string{
					"test": {"test"},
				},
			}).
		Return(nil)

	repoMock.EXPECT().CommitFiles("Add global script `test`", "manifest.json")

	if err := codebase.SetScript("test", []string{"test"}, true); err != nil {
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

	if err := codebase.MoveProject("test-1/something", "test-2/something"); !errors.Is(err, manifest.ErrNoProjectFound) {
		t.Errorf("wrong error (got: %s, want: %s)", err, manifest.ErrNoProjectFound)
	}

	// Move src doesn't exist relative path
	codebase.localPath = "test-1"
	if err := codebase.MoveProject("something", "test-2/something"); !errors.Is(err, manifest.ErrNoProjectFound) {
		t.Errorf("wrong error (got: %s, want: %s)", err, manifest.ErrNoProjectFound)
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

func TestCodebase_RmProject(t *testing.T) {
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

	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {Remote: "test-1.git"},
				"test/something-2": {Remote: "test-2.git"},
			},
		}, nil)

	// project doesn't exist
	if err := codebase.RmProject("test-1", false); !errors.Is(err, manifest.ErrNoProjectFound) {
		t.Fail()
	}

	// project exist but doesn't delete from disk
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {Remote: "test-1.git"},
				"test/something-2": {Remote: "test-2.git"},
			},
		}, nil)
	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-2": {Remote: "test-2.git"},
			},
		})
	repoMock.EXPECT().CommitFiles("Remove test/something-1", "manifest.json")

	if err := codebase.RmProject("test/something-1", false); err != nil {
		t.Fail()
	}

	// project exist but doesn't delete from disk (from local path)
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {Remote: "test-1.git"},
				"test/something-2": {Remote: "test-2.git"},
			},
		}, nil)
	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-2": {Remote: "test-2.git"},
			},
		})
	repoMock.EXPECT().CommitFiles("Remove test/something-1", "manifest.json")

	codebase.localPath = "test"
	if err := codebase.RmProject("something-1", false); err != nil {
		t.Fail()
	}

	// project exist but delete from disk
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {Remote: "test-1.git"},
				"test/something-2": {Remote: "test-2.git"},
			},
		}, nil)
	codebase.localPath = ""
	if err := os.MkdirAll(filepath.Join(path, "test", "something-2"), 0750); err != nil {
		t.FailNow()
	}
	manProviderMock.EXPECT().
		Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {Remote: "test-1.git"},
			},
		})
	repoMock.EXPECT().CommitFiles("Remove test/something-2", "manifest.json")

	if err := codebase.RmProject("test/something-2", true); err != nil {
		t.Fail()
	}

	// make sure project deleted
	if _, err := os.Stat(filepath.Join(path, "test", "something-2")); !os.IsNotExist(err) {
		t.Errorf("project not deleted")
	}
}

func TestCodebase_SetHook(t *testing.T) {
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

	// create codebase structure
	if err := os.MkdirAll(filepath.Join(path, "test", "something-1", ".git", "hooks"), 0750); err != nil {
		t.Fatal()
	}
	if err := os.MkdirAll(filepath.Join(path, "test", "something-2", ".git", "hooks"), 0750); err != nil {
		t.Fatal()
	}

	// set hook not in project
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {
					Remote: "test-1.git",
					Scripts: map[string][]string{
						"test-12": {"echo hello"},
					},
				},
				"test/something-2": {
					Remote: "test-2.git",
					Scripts: map[string][]string{
						"test-42": {"@global-42"},
					},
				},
			},
			Scripts: map[string][]string{
				"global-42": {"#/bin/sh", "echo hello from global"},
			},
		}, nil)
	if err := codebase.SetHook("test-12"); !errors.Is(err, manifest.ErrNoProjectFound) {
		t.Fail()
	}

	// set hook no script
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {
					Remote: "test-1.git",
					Scripts: map[string][]string{
						"test-12": {"echo hello"},
					},
				},
				"test/something-2": {
					Remote: "test-2.git",
					Scripts: map[string][]string{
						"test-42": {"@global-42"},
					},
				},
			},
			Scripts: map[string][]string{
				"global-42": {"#/bin/sh", "echo hello from global"},
			},
		}, nil)
	codebase.localPath = "test/something-1"
	if err := codebase.SetHook("test-111"); !errors.Is(err, manifest.ErrScriptNotFound) {
		t.Error(err)
	}

	// test set local hook working
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {
					Remote: "test-1.git",
					Scripts: map[string][]string{
						"test-12": {"echo hello"},
					},
				},
				"test/something-2": {
					Remote: "test-2.git",
					Scripts: map[string][]string{
						"test-42": {"@global-42"},
					},
				},
			},
			Scripts: map[string][]string{
				"global-42": {"#/bin/sh", "echo hello from global"},
			},
		}, nil)
	manProviderMock.EXPECT().Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
		Projects: map[string]manifest.Project{
			"test/something-1": {
				Remote: "test-1.git",
				Scripts: map[string][]string{
					"test-12": {"echo hello"},
				},
				Hook: "test-12",
			},
			"test/something-2": {
				Remote: "test-2.git",
				Scripts: map[string][]string{
					"test-42": {"@global-42"},
				},
			},
		},
		Scripts: map[string][]string{
			"global-42": {"#/bin/sh", "echo hello from global"},
		},
	})
	repoMock.EXPECT().CommitFiles("Set pre-commit hook `test-12` for test/something-1", manifestFile)
	if err := codebase.SetHook("test-12"); err != nil {
		t.Fail()
	}
	b, err := ioutil.ReadFile(filepath.Join(path, "test", "something-1", ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fail()
	}
	if string(b) != "echo hello" {
		t.Fatal()
	}

	// test set global hook working
	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.rootPath, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/something-1": {
					Remote: "test-1.git",
					Scripts: map[string][]string{
						"test-12": {"echo hello"},
					},
				},
				"test/something-2": {
					Remote: "test-2.git",
					Scripts: map[string][]string{
						"test-42": {"@global-42"},
					},
				},
			},
			Scripts: map[string][]string{
				"global-42": {"#/bin/sh", "echo hello from global"},
			},
		}, nil)
	codebase.localPath = "test/something-2"
	manProviderMock.EXPECT().Write(filepath.Join(codebase.rootPath, metaDir, manifestFile), manifest.Manifest{
		Projects: map[string]manifest.Project{
			"test/something-1": {
				Remote: "test-1.git",
				Scripts: map[string][]string{
					"test-12": {"echo hello"},
				},
			},
			"test/something-2": {
				Remote: "test-2.git",
				Scripts: map[string][]string{
					"test-42": {"@global-42"},
				},
				Hook: "test-42",
			},
		},
		Scripts: map[string][]string{
			"global-42": {"#/bin/sh", "echo hello from global"},
		},
	})
	repoMock.EXPECT().CommitFiles("Set pre-commit hook `test-42` for test/something-2", manifestFile)
	if err := codebase.SetHook("test-42"); err != nil {
		t.Fail()
	}
	b, err = ioutil.ReadFile(filepath.Join(path, "test", "something-2", ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fail()
	}
	if string(b) != "#/bin/sh\necho hello from global" {
		t.Fatal()
	}
}

func TestCodebase_Manifest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)

	codebase := &codebase{
		manProvider: manProviderMock,
		rootPath:    "/tmp/test",
	}

	val := manifest.Manifest{
		Projects: map[string]manifest.Project{"test": {}},
	}

	manProviderMock.EXPECT().Read(filepath.Join("/", "tmp", "test", metaDir, manifestFile)).
		Return(val, nil)

	if res, err := codebase.Manifest(); err != nil || !reflect.DeepEqual(res, val) {
		t.Fail()
	}
}
