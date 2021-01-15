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
	Add(remote, path string, config map[string]string) (manifest.Project, error)
	Sync(delete bool) (map[string]manifest.Project, map[string]manifest.Project, error)
	LocalPath() string
}

type codebase struct {
	// Full path to the codebase
	// Example: `/home/creekorful/Codebase`
	rootPath string
	// Local path inside the codebase
	// Example: `` or `Contributing/Debian`
	localPath string

	// The codebase Git repository
	repo repository.Repository

	// The repository provider (i.e the way we are opening, cloning Git repositories)
	repoProvider repository.Provider
	// The manifest provider (i.e the way we are reading/writing the manifest)
	manProvider manifest.Provider
}

func (codebase *codebase) Projects() (map[string]manifest.Project, error) {
	man, err := codebase.readManifest()
	if err != nil {
		return nil, err
	}

	return man.Projects, nil
}

func (codebase *codebase) Add(remote, path string, config map[string]string) (manifest.Project, error) {
	if path == "" {
		parts := strings.Split(remote, "/")
		path = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	if codebase.localPath != "" {
		path = filepath.Join(codebase.localPath, path)
	}

	man, err := codebase.readManifest()
	if err != nil {
		return manifest.Project{}, err
	}

	if man.Projects == nil {
		man.Projects = map[string]manifest.Project{}
	}

	// Make sure path is not taken
	if _, exist := man.Projects[path]; exist {
		return manifest.Project{}, fmt.Errorf("unable to add project %s: %w", remote, ErrPathTaken)
	}

	repo, err := codebase.repoProvider.Clone(remote, filepath.Join(codebase.rootPath, path))
	if err != nil {
		return manifest.Project{}, err
	}

	// Apply config
	for key, value := range config {
		if err := repo.SetConfig(key, value); err != nil {
			return manifest.Project{}, err
		}
	}

	// Update manifest
	man.Projects[path] = manifest.Project{
		Remote: remote,
		Config: config,
	}

	if err := codebase.writeManifest(man); err != nil {
		return manifest.Project{}, err
	}

	// Create commit
	if err := codebase.repo.CommitFiles(fmt.Sprintf("Add %s to %s", remote, path), manifestFile); err != nil {
		return manifest.Project{}, err
	}

	return man.Projects[path], nil
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
			for previousPath := range previousMan.Projects {
				if p == previousPath {
					found = true
					break
				}
			}

			// project not found in previous manifest
			if !found {
				addedProjects[p] = project

				// Clone the project (and don't break in case of error)
				_, _ = codebase.repoProvider.Clone(project.Remote, filepath.Join(codebase.rootPath, p))
			}

			if len(project.Config) > 0 {
				repo, err := codebase.repoProvider.Open(filepath.Join(codebase.rootPath, p))
				if err != nil {
					return nil, nil, err
				}

				// No matter what re-apply configuration to currently defined projects
				for key, value := range project.Config {
					if err := repo.SetConfig(key, value); err != nil {
						return nil, nil, err
					}
				}
			}
		}

		// Collect removed projects
		removedProjects := map[string]manifest.Project{}
		for previousPath, previousProject := range previousMan.Projects {
			found := false
			for p := range man.Projects {
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
					_ = os.RemoveAll(filepath.Join(codebase.rootPath, previousPath))
				}
			}
		}

		return addedProjects, removedProjects, nil
	}

	return nil, nil, nil
}

func (codebase *codebase) LocalPath() string {
	return codebase.localPath
}

func (codebase *codebase) readManifest() (manifest.Manifest, error) {
	man, err := codebase.manProvider.Read(filepath.Join(filepath.Join(codebase.rootPath, metaDir, manifestFile)))
	if err != nil {
		return manifest.Manifest{}, err
	}

	return man, nil
}

func (codebase *codebase) writeManifest(man manifest.Manifest) error {
	return codebase.manProvider.Write(filepath.Join(filepath.Join(codebase.rootPath, metaDir, manifestFile)), man)
}
