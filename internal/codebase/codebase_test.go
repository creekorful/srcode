package codebase

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/manifest_mock"
	"github.com/creekorful/srcode/internal/repository_mock"
	"github.com/golang/mock/gomock"
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
		repoProviderMock.EXPECT().
			Clone(test.repoRemote, filepath.Join(codebase.directory, test.localPath)).
			Return(nil, nil)

		manProviderMock.EXPECT().
			Read(filepath.Join(codebase.directory, metaDir, manifestFile)).
			Return(manifest.Manifest{}, nil)

		manProviderMock.EXPECT().
			Write(filepath.Join(codebase.directory, metaDir, manifestFile),
				manifest.Manifest{
					Projects: map[string]manifest.Project{
						test.localPath: {Remote: test.repoRemote},
					},
				}).
			Return(nil)

		repoMock.EXPECT().
			CommitFiles(fmt.Sprintf("Add %s to %s", test.repoRemote, test.localPath), "manifest.json").
			Return(nil)

		if err := codebase.Add(test.repoRemote, test.argPath); err != nil {
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

	if err := codebase.Add("git@github.com:test/test.git", "test/test"); !errors.Is(err, ErrPathTaken) {
		t.Fail()
	}
}

func TestCodebase_Sync(t *testing.T) {
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
		Read("/home/creekorful/.srcode/manifest.json").
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test/a/b": {Remote: "test.git"},
			},
		}, nil)

	repoMock.EXPECT().Pull("origin", "master").Return(nil)
	repoMock.EXPECT().Push("origin", "master").Return(nil)

	manProviderMock.EXPECT().
		Read("/home/creekorful/.srcode/manifest.json").
		Return(manifest.Manifest{
			Projects: map[string]manifest.Project{
				"test-12": {Remote: "test.git"},
			},
		}, nil)

	repoProviderMock.EXPECT().
		Clone("test.git", "/home/creekorful/test-12").
		Return(nil, nil)

	added, deleted, err := codebase.Sync()
	if err != nil {
		t.FailNow()
	}

	if len(added) != 1 {
		t.Fail()
	}
	if len(deleted) != 1 {
		t.Fail()
	}
}
