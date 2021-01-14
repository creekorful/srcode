package codebase

import (
	"errors"
	"github.com/creekorful/srcode/internal/repository_mock"
	"github.com/golang/mock/gomock"
	"os"
	"path/filepath"
	"testing"
)

func TestProvider_New(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	provider := provider{repoProvider: repoProviderMock}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Repository provider fails
	repoProviderMock.EXPECT().New(filepath.Join(targetDir, metaDir)).Return(nil, errors.New("test error"))
	if _, err := provider.New(targetDir); err == nil {
		t.Error(err)
	}

	repoProviderMock.EXPECT().New(filepath.Join(targetDir, metaDir)).Return(nil, nil)
	val, err := provider.New(targetDir)
	if err != nil {
		t.Fatal(err)
	}

	if val.(*codebase).directory != targetDir {
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

	if _, err := provider.New(targetDir); !errors.Is(err, ErrCodebaseAlreadyExist) {
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

	if val.(*codebase).directory != targetDir {
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

	if _, err := provider.Clone("something", targetDir); !errors.Is(err, ErrCodebaseAlreadyExist) {
		t.Fail()
	}
}

func TestProvider_Clone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repoProviderMock := repository_mock.NewMockProvider(mockCtrl)

	provider := provider{repoProvider: repoProviderMock}

	targetDir := filepath.Join(t.TempDir(), "test-directory")

	// Cloning has fail
	repoProviderMock.EXPECT().
		Clone("test-remote", filepath.Join(targetDir, metaDir)).
		Return(nil, errors.New("test error"))
	if _, err := provider.Clone("test-remote", targetDir); err == nil {
		t.Fail()
	}

	repoProviderMock.EXPECT().
		Clone("test-remote", filepath.Join(targetDir, metaDir)).
		Return(nil, nil)
	val, err := provider.Clone("test-remote", targetDir)
	if err != nil {
		t.Fail()
	}

	if val.(*codebase).directory != targetDir {
		t.Fail()
	}
}
