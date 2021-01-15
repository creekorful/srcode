package codebase

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/fs"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrCodebaseAlreadyExist is returned when a codebase already exist in current directory
	ErrCodebaseAlreadyExist = errors.New("a codebase already exist")
	// ErrCodebaseNotExist is returned when no codebase is found in current or parent directories
	ErrCodebaseNotExist = errors.New("no codebase found in current or parent directories")

	// DefaultProvider is the default codebase provider
	DefaultProvider = &provider{
		repoProvider:     repository.DefaultProvider,
		manifestProvider: &manifest.JSONProvider{},
	}
)

const (
	metaDir      = ".srcode"
	manifestFile = "manifest.json"
)

// Provider is something that allows to Init, Open, or Clone a Codebase
type Provider interface {
	Init(path, remote string) (Codebase, error)
	Open(path string) (Codebase, error)
	Clone(url, path string) (Codebase, error)
}

type provider struct {
	repoProvider     repository.Provider
	manifestProvider manifest.Provider
}

func (provider *provider) Init(path, remote string) (Codebase, error) {
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
	repo, err := provider.repoProvider.Init(filepath.Join(path, metaDir))
	if err != nil {
		return nil, fmt.Errorf("error while creating codebase at %s: %w", path, err)
	}

	// create the meta file
	b, err := json.Marshal(manifest.Manifest{})
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(filepath.Join(path, metaDir, manifestFile), b, 0640); err != nil {
		return nil, fmt.Errorf("error while creating manifest: %w", err)
	}

	// create the README.md
	readmeMd := getReadmeMD(remote)
	if err := ioutil.WriteFile(filepath.Join(path, metaDir, "README.md"), []byte(readmeMd), 0640); err != nil {
		return nil, fmt.Errorf("error while writing README.md: %w", err)
	}

	// Create initial commit
	if err := repo.CommitFiles("Initial commit", manifestFile, "README.md"); err != nil {
		return nil, err
	}

	// Set remote if provided
	if remote != "" {
		if err := repo.AddRemote("origin", remote); err != nil {
			return nil, err
		}
	}

	return &codebase{
		rootPath:     path,
		repoProvider: provider.repoProvider,
		repo:         repo,
		manProvider:  provider.manifestProvider,
	}, nil
}

func (provider *provider) Open(path string) (Codebase, error) {
	rootPath := path
	localPath := ""

	// Easier case: we are at codebase root
	exist, err := codebaseExists(rootPath)
	if err != nil {
		return nil, err
	}

	if !exist {
		// Otherwise lookup parent directories
		parentDirs := fs.GetParentDirs(rootPath)
		for _, dir := range parentDirs {
			found, err := codebaseExists(dir)
			if err != nil {
				continue // TODO fails?
			}

			if found {
				localPath = strings.TrimPrefix(rootPath, dir+"/")
				rootPath = dir
				exist = true
				break
			}
		}
	}

	if !exist {
		return nil, fmt.Errorf("error while opening codebase at %s: %w", rootPath, ErrCodebaseNotExist)
	}

	repo, err := provider.repoProvider.Open(filepath.Join(rootPath, metaDir))
	if err != nil {
		return nil, fmt.Errorf("error while opening codebase at %s: %w", path, err)
	}

	return &codebase{
		rootPath:     rootPath,
		localPath:    localPath,
		repoProvider: provider.repoProvider,
		repo:         repo,
		manProvider:  provider.manifestProvider,
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

	codebase := &codebase{
		rootPath:     path,
		repoProvider: provider.repoProvider,
		repo:         repo,
		manProvider:  provider.manifestProvider,
	}

	projects, err := codebase.Projects()
	if err != nil {
		return nil, err
	}

	// Clone back project & configure them if needed
	for projectPath, project := range projects {
		repo, err := provider.repoProvider.Clone(project.Remote, filepath.Join(path, projectPath))
		if err != nil {
			return nil, err
		}

		for key, value := range project.Config {
			if err := repo.SetConfig(key, value); err != nil {
				return nil, err
			}
		}
	}

	return codebase, nil
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

func getReadmeMD(remote string) string {
	b := strings.Builder{}

	b.WriteString("# dot-srcode\n\n")
	b.WriteString("This repository contains a custom .srcode configuration.\n")
	b.WriteString("See [this repository](https://github.com/creekorful/srcode) for more details.\n\n")

	if remote != "" {
		b.WriteString("## How to use it\n\n")
		b.WriteString("One can restore this codebase by issuing the following command:\n\n")
		b.WriteString("```\n")
		b.WriteString(fmt.Sprintf("$ srcode clone %s code\n", remote))
		b.WriteString("```\n")
	}

	return b.String()
}
