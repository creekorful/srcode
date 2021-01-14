package repository

//go:generate mockgen -destination=../repository_mock/repository_mock.go -package=repository_mock . Repository

type Repository interface {
	CommitFiles(message string, files ...string) error
}
