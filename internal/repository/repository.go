package repository

import "os/exec"

//go:generate mockgen -destination=../repository_mock/repository_mock.go -package=repository_mock . Repository

type Repository interface {
	CommitFiles(message string, files ...string) error
	Push(repo, refspec string) error
	Pull(repo, refspec string) error
	AddRemote(name, url string) error
	Config(key string) (string, error)
	SetConfig(key, value string) error
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

func (gwr *gitWrapperRepository) Push(repo, refspec string) error {
	_, err := gwr.execWithOutput("push", repo, refspec)
	return err
}

func (gwr *gitWrapperRepository) Pull(repo, refspec string) error {
	_, err := gwr.execWithOutput("pull", "--rebase", repo, refspec)
	return err
}

func (gwr *gitWrapperRepository) AddRemote(name, url string) error {
	_, err := gwr.execWithOutput("remote", "add", name, url)
	return err
}

func (gwr *gitWrapperRepository) Config(key string) (string, error) {
	return gwr.execWithOutput("config", key)
}

func (gwr *gitWrapperRepository) SetConfig(key, value string) error {
	_, err := gwr.execWithOutput("config", key, value)
	return err
}

func (gwr *gitWrapperRepository) execWithOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = gwr.path

	return execWithOutput(cmd)
}
