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
	"testing"
)

func TestCodebase_Projects(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projects := map[string]manifest.Project{
		"test/15": {Remote: "test"},
	}

	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	manProviderMock.EXPECT().
		Read("test-dir/.srcode/manifest.json").
		Return(manifest.Manifest{
			Projects: projects,
		}, nil)

	codebase := &codebase{
		manProvider: manProviderMock,
		directory:   "test-dir",
	}

	projects, err := codebase.Projects()
	if err != nil {
		t.FailNow()
	}

	if !reflect.DeepEqual(projects, projects) {
		t.FailNow()
	}
}

func TestCodebase_Add(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)
	manProviderMock := manifest_mock.NewMockProvider(mockCtrl)
	repoMock := repository_mock.NewMockRepository(mockCtrl)

	codebase := &codebase{
		repoProvider: repoProviderMock,
		repo:         repoMock,
		manProvider:  manProviderMock,
		directory:    "/home/creekorful",
	}

	type test struct {
		repoRemote string

		localPath string
		argPath   string
	}
	tests := []test{
		{
			repoRemote: "git@github.com:creekorful/test.git",
			localPath:  "test",
			argPath:    "",
		},
		{
			repoRemote: "git@github.com:creekorful/test.git",
			localPath:  "Test-folder/Another/test",
			argPath:    "Test-folder/Another/test",
		},
	}

	for _, test := range tests {
		currentRepoMock := repository_mock.NewMockRepository(mockCtrl)

		repoProviderMock.EXPECT().
			Clone(test.repoRemote, filepath.Join(codebase.directory, test.localPath)).
			Return(currentRepoMock, nil)

		manProviderMock.EXPECT().
			Read(filepath.Join(codebase.directory, metaDir, manifestFile)).
			Return(manifest.Manifest{}, nil)

		manProviderMock.EXPECT().
			Write(filepath.Join(codebase.directory, metaDir, manifestFile),
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
		directory:    "/home/creekorful",
	}

	manProviderMock.EXPECT().
		Read(filepath.Join(codebase.directory, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/test": {Remote: ""},
			},
		}, nil)

	if _, err := codebase.Add("git@github.com:test/test.git", "test/test", nil); !errors.Is(err, ErrPathTaken) {
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
		directory:    dir,
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
			},
		}, nil)

	repoMock.EXPECT().Pull("origin", "master").Return(nil)
	repoMock.EXPECT().Push("origin", "master").Return(nil)

	manProviderMock.EXPECT().
		Read(filepath.Join(dir, metaDir, manifestFile)).
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test-12": {Remote: "test.git"},
			},
		}, nil)

	repoProviderMock.EXPECT().
		Clone("test.git", filepath.Join(dir, "test-12")).
		Return(nil, nil)

	added, deleted, err := codebase.Sync(true)
	if err != nil {
		t.FailNow()
	}

	if len(added) != 1 {
		t.Fail()
	}
	if len(deleted) != 1 {
		t.Fail()
	}

	// make sure we have deleted the directory
	if _, err := os.Stat(filepath.Join(dir, "test", "a", "b")); !os.IsNotExist(err) {
		t.Fail()
	}
}
