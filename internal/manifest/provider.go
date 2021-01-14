package manifest

//go:generate mockgen -destination=../manifest_mock/manifest_mock.go -package=manifest_mock . Provider

type Provider interface {
	Read(path string) (Manifest, error)
	Write(path string, manifest Manifest) error
}
