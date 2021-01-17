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
	"reflect"
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

	codebase := &codebase{
		manProvider: manProviderMock,
		rootPath:    "test-dir",
	}

	projects, err := codebase.Projects()
	if err != nil {
		t.FailNow()
	}

	if !reflect.DeepEqual(projects, expectedProjects) {
		t.FailNow()
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
