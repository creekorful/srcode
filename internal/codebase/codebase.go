package codebase

//go:generate mockgen -destination=../codebase_mock/codebase_mock.go -package=codebase_mock . Codebase

import "github.com/creekorful/srcode/internal/repository"

type Codebase interface {
	Projects() []Project
}

type codebase struct {
	repo      repository.Repository
	directory string
}

func (codebase *codebase) Projects() []Project {
	return nil
}
