package repository

import (
	"github.com/creekorful/srcode/internal/cmd"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultProvider is the default Git provider
var DefaultProvider = &gitWrapperProvider{}

// Provider is something that allows to Init, Open, or Clone a Repository
type Provider interface {
	Init(path string) (Repository, error)
	Open(path string) (Repository, error)
	Clone(url, path string) (Repository, error)
	Exists(path string) bool
}

type gitWrapperProvider struct {
}

func (gwp *gitWrapperProvider) Init(path string) (Repository, error) {
	command := exec.Command("git", "init", path)

	if _, err := cmd.ExecWithOutput(command); err != nil {
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
	command := exec.Command("git", "clone", url, path)

	if _, err := cmd.ExecWithOutput(command); err != nil {
		return nil, err
	}

	return &gitWrapperRepository{path: path}, nil
}

func (gwp *gitWrapperProvider) Exists(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
