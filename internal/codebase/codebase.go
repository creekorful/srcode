package codebase

//go:generate mockgen -destination=../codebase_mock/codebase_mock.go -package=codebase_mock . Codebase,Provider

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/cmd"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository"
	"golang.org/x/sync/errgroup"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	// ErrPathTaken is returned when a project already exist at given path
	ErrPathTaken = errors.New("a project already exist at given path")
	// ErrNoProjectFound is returned when no project exist at given path
	ErrNoProjectFound = errors.New("no project exist at given path")
	// ErrCommandNotFound is returned when given command is not found
	ErrCommandNotFound = errors.New("no command with the name found")
)

// Codebase is a collection of projects
type Codebase interface {
	Projects() (map[string]manifest.Project, error)
	Add(remote, path string, config map[string]string) (manifest.Project, error)
	Sync(delete bool, addedChan chan<- ProjectEntry, deletedChan chan<- ProjectEntry) error
	LocalPath() string
	Run(command string) (string, error)
	BulkGIT(args []string, out chan<- string) error
	SetCommand(name, command string, global bool) error
	MoveProject(oldPath, newPath string) error
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

func (codebase *codebase) Sync(delete bool, addedChan chan<- ProjectEntry, deletedChan chan<- ProjectEntry) error {
	defer func() {
		if addedChan != nil {
			close(addedChan)
		}
		if deletedChan != nil {
			close(deletedChan)
		}
	}()

	// read manifest
	previousMan, err := codebase.readManifest()
	if err != nil {
		return err
	}

	// pull & push
	// Allow to fail because may fail if not already pushed (todo better)
	_ = codebase.repo.Pull("origin", "master")

	if err := codebase.repo.Push("origin", "master"); err != nil {
		return err
	}

	// read again manifest
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	// apply changes (if there are)
	if !reflect.DeepEqual(previousMan, man) {
		g := errgroup.Group{}

		// Collect new projects
		for p, project := range man.Projects {
			p := p
			project := project
			found := false

			for previousPath := range previousMan.Projects {
				if p == previousPath {
					found = true
					break
				}
			}

			g.Go(func() error {
				// project not found in previous manifest
				if !found {
					if addedChan != nil {
						addedChan <- ProjectEntry{
							Path:    p,
							Project: project,
						}
					}

					// Clone the project (and don't break in case of error)
					_, _ = codebase.repoProvider.Clone(project.Remote, filepath.Join(codebase.rootPath, p))
				}

				// (Re-)Apply the configuration
				if len(project.Config) > 0 {
					repo, err := codebase.repoProvider.Open(filepath.Join(codebase.rootPath, p))
					if err != nil {
						return err
					}

					// No matter what re-apply configuration to currently defined projects
					for key, value := range project.Config {
						if err := repo.SetConfig(key, value); err != nil {
							return err
						}
					}
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		// Collect removed projects
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
				if deletedChan != nil {
					deletedChan <- ProjectEntry{
						Path:    previousPath,
						Project: previousProject,
					}
				}

				// Remove from disk (and don't break in case of error)
				if delete {
					_ = os.RemoveAll(filepath.Join(codebase.rootPath, previousPath))
				}
			}
		}
	}

	return nil
}

func (codebase *codebase) LocalPath() string {
	return codebase.localPath
}

func (codebase *codebase) Run(command string) (string, error) {
	// Retrieve manifest
	man, err := codebase.readManifest()
	if err != nil {
		return "", err
	}

	// Fails if we are not in a project directory
	project, exist := man.Projects[codebase.localPath]
	if !exist {
		return "", ErrNoProjectFound
	}

	// Check if command is defined locally
	cmdStr, exist := project.Commands[command]
	if !exist {
		return "", fmt.Errorf("error while running command %s: %w", command, ErrCommandNotFound)
	}

	// It's a global alias
	if strings.HasPrefix(cmdStr, "@") {
		cmdStr, exist = man.Commands[strings.TrimPrefix(cmdStr, "@")]
		if !exist {
			return "", fmt.Errorf("error while running command %s: %w", command, ErrCommandNotFound)
		}
	}

	// Finally execute the command
	parts := strings.Split(cmdStr, " ")
	return cmd.ExecWithOutput(exec.Command(parts[0], parts[1:]...))
}

func (codebase *codebase) BulkGIT(args []string, out chan<- string) error {
	defer func() {
		if out != nil {
			close(out)
		}
	}()

	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	for path := range man.Projects {
		repo, err := codebase.repoProvider.Open(filepath.Join(codebase.rootPath, path))
		if err != nil {
			return err
		}

		res, err := repo.RawCmd(args)
		if err != nil {
			return err
		}

		if out != nil {
			out <- res
		}
	}

	return nil
}

func (codebase *codebase) SetCommand(name, command string, global bool) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	// This is a global command
	if global {
		if man.Commands == nil {
			man.Commands = map[string]string{}
		}

		man.Commands[name] = command

		if err := codebase.writeManifest(man); err != nil {
			return err
		}

		if err := codebase.repo.CommitFiles(
			fmt.Sprintf("Add command `%s`: `%s`", name, command),
			manifestFile,
		); err != nil {
			return err
		}

		return nil
	}

	project, exist := man.Projects[codebase.localPath]

	if !exist {
		return ErrNoProjectFound
	}

	if project.Commands == nil {
		project.Commands = map[string]string{}
	}

	project.Commands[name] = command
	man.Projects[codebase.localPath] = project

	if err := codebase.writeManifest(man); err != nil {
		return err
	}

	if err := codebase.repo.CommitFiles(
		fmt.Sprintf("Add command `%s`: `%s`", name, command),
		manifestFile,
	); err != nil {
		return err
	}

	return nil
}

func (codebase *codebase) MoveProject(oldPath, newPath string) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	oldPath = filepath.Join(codebase.localPath, oldPath)
	newPath = filepath.Join(codebase.localPath, newPath)

	if _, exist := man.Projects[oldPath]; !exist {
		return ErrNoProjectFound
	}

	if _, exist := man.Projects[newPath]; exist {
		return ErrPathTaken
	}

	oldDiskPath := filepath.Join(codebase.rootPath, oldPath)
	newDiskPath := filepath.Join(codebase.rootPath, newPath)

	// Create any missing parent directories
	if parentDir := filepath.Dir(newDiskPath); parentDir != "." {
		if err := os.MkdirAll(parentDir, 0750); err != nil {
			return nil
		}
	}

	// Finally move the project
	if err := os.Rename(oldDiskPath, newDiskPath); err != nil {
		return err
	}

	// Update the manifest
	project := man.Projects[oldPath]
	delete(man.Projects, oldPath)
	man.Projects[newPath] = project

	if err := codebase.writeManifest(man); err != nil {
		return err
	}

	msg := fmt.Sprintf("Moved %s from %s to %s", project.Remote, oldPath, newPath)
	if err := codebase.repo.CommitFiles(msg, manifestFile); err != nil {
		return err
	}

	return nil
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
