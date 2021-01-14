package repository

import "os/exec"

//go:generate mockgen -destination=../repository_mock/repository_mock.go -package=repository_mock . Repository

type Repository interface {
	CommitFiles(message string, files ...string) error
}

type gitWrapperRepository struct {
	path string
}

func (gwr *gitWrapperRepository) CommitFiles(message string, files ...string) error {
	if _, err := gwr.execWithOutput(append([]string{"add"}, files...)...); err != nil {
		return err
	}

	if _, err := gwr.execWithOutput("commit", "-m", message); err != nil {
		return err
	}

	return nil
}

func (gwr *gitWrapperRepository) execWithOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = gwr.path

	return execWithOutput(cmd)
}
