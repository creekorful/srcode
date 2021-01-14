package codebase

//go:generate mockgen -destination=../codebase_mock/codebase_mock.go -package=codebase_mock . Codebase

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var ErrPathTaken = errors.New("a project already exist at given path")

type Codebase interface {
	Projects() (map[string]manifest.Project, error)
	Add(remote, path string) error

	Sync(delete bool) (map[string]manifest.Project, map[string]manifest.Project, error)
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

func (codebase *codebase) Sync(delete bool) (map[string]manifest.Project, map[string]manifest.Project, error) {
	// read manifest
	previousMan, err := codebase.readManifest()
	if err != nil {
		return nil, nil, err
	}

	// pull & push
	// Allow to fail because may fail if not already pushed (todo better)
	_ = codebase.repo.Pull("origin", "master")

	if err := codebase.repo.Push("origin", "master"); err != nil {
		return nil, nil, err
	}

	// read again manifest
	man, err := codebase.readManifest()
	if err != nil {
		return nil, nil, err
	}

	// apply changes (if there are)
	if !reflect.DeepEqual(previousMan, man) {
		// Collect new projects
		addedProjects := map[string]manifest.Project{}
		for p, project := range man.Projects {
			found := false
			for previousPath, _ := range previousMan.Projects {
				if p == previousPath {
					found = true
					break
				}
			}

			// project not found in previous manifest
			if !found {
				addedProjects[p] = project

				// Clone the project (and don't break in case of error)
				_, _ = codebase.repoProvider.Clone(project.Remote, filepath.Join(codebase.directory, p))
			}
		}

		// Collect removed projects
		removedProjects := map[string]manifest.Project{}
		for previousPath, previousProject := range previousMan.Projects {
			found := false
			for p, _ := range man.Projects {
				if p == previousPath {
					found = true
					break
				}
			}

			// project not found in current manifest
			if !found {
				removedProjects[previousPath] = previousProject

				// Remove from disk (and don't break in case of error)
				if delete {
					_ = os.RemoveAll(filepath.Join(codebase.directory, previousPath))
				}
			}
		}

		return addedProjects, removedProjects, nil
	}

	return nil, nil, nil
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
