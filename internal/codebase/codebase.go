package codebase

//go:generate mockgen -destination=../codebase_mock/codebase_mock.go -package=codebase_mock . Codebase

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository"
	"path/filepath"
	"strings"
)

var ErrPathTaken = errors.New("a project already exist at given path")

type Codebase interface {
	Projects() (map[string]manifest.Project, error)
	Add(remote, path string) error
}

type codebase struct {
	directory    string
	repoProvider repository.Provider
	repo         repository.Repository
	manProvider  manifest.Provider
}

func (codebase *codebase) Projects() (map[string]manifest.Project, error) {
	man, err := codebase.readManifest()
	if err != nil {
		return nil, err
	}

	return man.Projects, nil
}

func (codebase *codebase) Add(remote, path string) error {
	if path == "" {
		parts := strings.Split(remote, "/")
		path = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	if man.Projects == nil {
		man.Projects = map[string]manifest.Project{}
	}

	// Make sure path is not taken
	if _, exist := man.Projects[path]; exist {
		return fmt.Errorf("unable to add project %s: %w", remote, ErrPathTaken)
	}

	if _, err := codebase.repoProvider.Clone(remote, filepath.Join(codebase.directory, path)); err != nil {
		return err
	}

	// Update manifest
	man.Projects[path] = manifest.Project{
		Remote: remote,
	}

	if err := codebase.writeManifest(man); err != nil {
		return err
	}

	// Create commit
	if err := codebase.repo.CommitFiles(fmt.Sprintf("Add %s to %s", remote, path), manifestFile); err != nil {
		return err
	}

	return nil
}

func (codebase *codebase) readManifest() (manifest.Manifest, error) {
	man, err := codebase.manProvider.Read(filepath.Join(filepath.Join(codebase.directory, metaDir, manifestFile)))
	if err != nil {
		return manifest.Manifest{}, err
	}

	return man, nil
}

func (codebase *codebase) writeManifest(man manifest.Manifest) error {
	return codebase.manProvider.Write(filepath.Join(filepath.Join(codebase.directory, metaDir, manifestFile)), man)
}
