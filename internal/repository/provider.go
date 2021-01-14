package repository

//go:generate mockgen -destination=../repository_mock/provider_mock.go -package=repository_mock . Provider

type Provider interface {
	New(path string) (Repository, error)
	Open(path string) (Repository, error)
	Clone(url, path string) (Repository, error)
}
