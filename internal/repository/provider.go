package repository

import (
	"os"
	"os/exec"
)

//go:generate mockgen -destination=../repository_mock/provider_mock.go -package=repository_mock . Provider

var DefaultProvider = &gitWrapperProvider{}

type Provider interface {
	New(path string) (Repository, error)
	Open(path string) (Repository, error)
	Clone(url, path string) (Repository, error)
}

type gitWrapperProvider struct {
}

func (gwp *gitWrapperProvider) New(path string) (Repository, error) {
	cmd := exec.Command("git", "init", path)

	if _, err := execWithOutput(cmd); err != nil {
		return nil, err
	}

	return &gitWrapperRepository{path: path}, nil
}

func (gwp *gitWrapperProvider) Open(path string) (Repository, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &gitWrapperRepository{path: path}, nil
}

func (gwp *gitWrapperProvider) Clone(url, path string) (Repository, error) {
	cmd := exec.Command("git", "clone", url, path)

	if _, err := execWithOutput(cmd); err != nil {
		return nil, err
	}

	return &gitWrapperRepository{path: path}, nil
}
