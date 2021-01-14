package codebase

//go:generate mockgen -destination=../codebase_mock/codebase_mock.go -package=codebase_mock . Codebase

import (
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/creekorful/srcode/internal/repository"
)

type Codebase interface {
	Projects() map[string]manifest.Project
}

type codebase struct {
	repo      repository.Repository
	directory string
	manifest  manifest.Manifest
}

func (codebase *codebase) Projects() map[string]manifest.Project {
	return codebase.manifest.Projects
}
