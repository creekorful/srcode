package codebase

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/repository"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	ErrCodebaseAlreadyExist = errors.New("a codebase already exist")
	ErrCodebaseNotExist     = errors.New("no codebase found")
)

const (
	metaDir  = ".srcode"
	manifest = "manifest.json"
)

type Provider interface {
	New(path string) (Codebase, error)
	Open(path string) (Codebase, error)
	Clone(url, path string) (Codebase, error)
}

type provider struct {
	repoProvider repository.Provider
}

func (provider *provider) New(path string) (Codebase, error) {
	exist, err := codebaseExists(path)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("error while creating codebase at %s: %w", path, ErrCodebaseAlreadyExist)
	}

	// create directories
	if err := os.MkdirAll(filepath.Join(path, metaDir), 0750); err != nil {
		return nil, err
	}

	// create git (meta) directory
	repo, err := provider.repoProvider.New(filepath.Join(path, metaDir))
	if err != nil {
		return nil, fmt.Errorf("error while creating codebase at %s: %w", path, err)
	}

	// create the meta file
	b, err := json.Marshal(Manifest{})
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(filepath.Join(path, metaDir, manifest), b, 0640); err != nil {
		return nil, fmt.Errorf("error while creating manifest: %w", err)
	}

	// Create initial commit
	if err := repo.CommitFiles("Initial commit", manifest); err != nil {
		return nil, err
	}

	return &codebase{
		repo:      repo,
		directory: path,
	}, nil
}

func (provider *provider) Open(path string) (Codebase, error) {
	exist, err := codebaseExists(path)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("error while opening codebase at %s: %w", path, ErrCodebaseNotExist)
	}

	repo, err := provider.repoProvider.Open(filepath.Join(path, metaDir))
	if err != nil {
		return nil, fmt.Errorf("error while opening codebase at %s: %w", path, err)
	}

	return &codebase{
		repo:      repo,
		directory: path,
	}, nil
}

func (provider *provider) Clone(url, path string) (Codebase, error) {
	exist, err := codebaseExists(path)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("error while cloning codebase at %s: %w", path, ErrCodebaseAlreadyExist)
	}

	repo, err := provider.repoProvider.Clone(url, filepath.Join(path, metaDir))
	if err != nil {
		return nil, fmt.Errorf("error while cloning codebase: %w", err)
	}

	// TODO install projects

	return &codebase{
		repo:      repo,
		directory: path,
	}, nil
}

func codebaseExists(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, metaDir))
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}
