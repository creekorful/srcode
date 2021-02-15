package codebase

//go:generate mockgen -destination=../codebase_mock/codebase_mock.go -package=codebase_mock . Codebase,Provider

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository"
	"github.com/fatih/color"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	// ErrPathTaken is returned when a project already exist at given path
	ErrPathTaken = errors.New("a project already exist at given path")
)

// ProjectEntry map a codebase project entry (i.e the project alongside his codebase local path)
// and optionally the linked Git repository
type ProjectEntry struct {
	Path       string
	Project    manifest.Project
	Repository repository.Repository
}

// Codebase is a collection of projects
type Codebase interface {
	Projects() (map[string]ProjectEntry, error)
	Manifest() (manifest.Manifest, error)
	Add(remote, path string, config map[string]string) (manifest.Project, error)
	Sync(delete bool, addedChan chan<- ProjectEntry, deletedChan chan<- ProjectEntry) error
	LocalPath() string
	Run(scriptName string, args []string, writer io.Writer) error
	BulkGIT(args []string, writer io.Writer) error
	SetScript(name string, script []string, global bool) error
	MoveProject(oldPath, newPath string) error
	RmProject(path string, delete bool) error
	SetHook(scriptName string) error
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

func (codebase *codebase) Projects() (map[string]ProjectEntry, error) {
	man, err := codebase.readManifest()
	if err != nil {
		return nil, err
	}

	entries := map[string]ProjectEntry{}

	for path, project := range man.Projects {
		repo, err := codebase.repoProvider.Open(filepath.Join(codebase.rootPath, path))
		if err != nil {
			return nil, err
		}

		entries[path] = ProjectEntry{
			Path:       path,
			Project:    project,
			Repository: repo,
		}
	}

	return entries, nil
}

func (codebase *codebase) Manifest() (manifest.Manifest, error) {
	return codebase.readManifest()
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
	_ = codebase.repo.Pull("origin", "main")

	if err := codebase.repo.Push("origin", "main"); err != nil {
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

				// (Re-)Configure the project
				if err := codebase.configureProject(man, p); err != nil {
					return err
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

func (codebase *codebase) Run(scriptName string, args []string, writer io.Writer) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	scriptVal, err := man.GetScript(codebase.localPath, scriptName)
	if err != nil {
		return fmt.Errorf("error while running script %s: %w", scriptName, err)
	}

	// Finally execute the script
	file, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return err
	}
	path := file.Name()

	defer os.Remove(path)

	if _, err := io.WriteString(file, strings.Join(scriptVal, "\n")); err != nil {
		return err
	}

	cmdArgs := []string{path}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("sh", cmdArgs...)
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}

func (codebase *codebase) BulkGIT(args []string, writer io.Writer) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	sepStyle := color.New(color.Bold, color.FgHiYellow).Sprint("===")
	pathStyle := color.New(color.Bold, color.FgHiWhite)

	for path := range man.Projects {
		repo, err := codebase.repoProvider.Open(filepath.Join(codebase.rootPath, path))
		if err != nil {
			return err
		}

		_, _ = io.WriteString(writer, fmt.Sprintf("%s /%s %s\n\n", sepStyle, pathStyle.Sprint(path), sepStyle))

		if err := repo.RawCmd(args, writer); err != nil {
			return err
		}

		_, _ = io.WriteString(writer, "\n")
	}

	return nil
}

func (codebase *codebase) SetScript(name string, script []string, global bool) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	// This is a global script
	if global {
		if man.Scripts == nil {
			man.Scripts = map[string][]string{}
		}

		man.Scripts[name] = script

		if err := codebase.writeManifest(man); err != nil {
			return err
		}

		if err := codebase.repo.CommitFiles(
			fmt.Sprintf("Add global script `%s`", name),
			manifestFile,
		); err != nil {
			return err
		}

		return nil
	}

	project, exist := man.Projects[codebase.localPath]

	if !exist {
		return manifest.ErrNoProjectFound
	}

	if project.Scripts == nil {
		project.Scripts = map[string][]string{}
	}

	project.Scripts[name] = script
	man.Projects[codebase.localPath] = project

	if err := codebase.writeManifest(man); err != nil {
		return err
	}

	if err := codebase.repo.CommitFiles(
		fmt.Sprintf("Add script `%s` to %s", name, codebase.localPath),
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
		return manifest.ErrNoProjectFound
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

func (codebase *codebase) RmProject(path string, shouldDelete bool) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	path = filepath.Join(codebase.localPath, path)

	if _, exist := man.Projects[path]; !exist {
		return manifest.ErrNoProjectFound
	}

	delete(man.Projects, path)

	if err := codebase.writeManifest(man); err != nil {
		return err
	}

	if err := codebase.repo.CommitFiles(fmt.Sprintf("Remove %s", path), manifestFile); err != nil {
		return err
	}

	if shouldDelete {
		if err := os.RemoveAll(filepath.Join(codebase.rootPath, path)); err != nil {
			return err
		}
	}

	return nil
}

func (codebase *codebase) SetHook(scriptName string) error {
	man, err := codebase.readManifest()
	if err != nil {
		return err
	}

	script, err := man.GetScript(codebase.localPath, scriptName)
	if err != nil {
		return fmt.Errorf("error while setting hook %s: %w", scriptName, err)
	}

	// copy the script to .git/hooks directory
	f, err := os.OpenFile(filepath.Join(codebase.rootPath, codebase.localPath, ".git", "hooks", "pre-push"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return err
	}
	defer f.Close()

	// write the content
	if _, err := io.WriteString(f, strings.Join(script, "\n")); err != nil {
		return err
	}

	// Update the manifest
	project, exists := man.Projects[codebase.localPath]
	if !exists {
		return manifest.ErrNoProjectFound // should not happen there
	}
	project.Hook = scriptName
	man.Projects[codebase.localPath] = project

	if err := codebase.writeManifest(man); err != nil {
		return err
	}

	// Commit the changes
	if err := codebase.repo.CommitFiles(fmt.Sprintf("Set pre-push hook `%s` for %s", scriptName, codebase.localPath), manifestFile); err != nil {
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

func (codebase *codebase) configureProject(man manifest.Manifest, path string) error {
	project, exist := man.Projects[path]
	if !exist {
		return manifest.ErrNoProjectFound
	}

	// (Re-)Apply the configuration
	if len(project.Config) > 0 {
		repo, err := codebase.repoProvider.Open(filepath.Join(codebase.rootPath, path))
		if err != nil {
			return err
		}

		for key, value := range project.Config {
			if err := repo.SetConfig(key, value); err != nil {
				return err
			}
		}
	}

	// Apply hook if any
	if project.Hook != "" {
		script, err := man.GetScript(path, project.Hook)
		if err == nil {
			f, err := os.OpenFile(filepath.Join(codebase.rootPath, path, ".git", "hooks", "pre-push"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
			if err != nil {
				return err
			}
			defer f.Close()

			// write the content
			if _, err := io.WriteString(f, strings.Join(script, "\n")); err != nil {
				return err
			}
		}
	}

	return nil
}
