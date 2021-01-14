package codebase

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/repository"
	"os"
	"path/filepath"
)

var (
	ErrCodebaseAlreadyExist = errors.New("a codebase already exist")
	ErrCodebaseNotExist     = errors.New("no codebase found")
)

const metaDir = ".srcode"

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

	repo, err := provider.repoProvider.New(filepath.Join(path, metaDir))
	if err != nil {
		return nil, fmt.Errorf("error while creating codebase at %s: %w", path, err)
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
