package manifest

import (
	"encoding/json"
	"io/ioutil"
)

//go:generate mockgen -destination=../manifest_mock/manifest_mock.go -package=manifest_mock . Provider

// Provider is something that allows to Read or Write a Manifest
type Provider interface {
	Read(path string) (Manifest, error)
	Write(path string, manifest Manifest) error
}

// JSONProvider is a provider that use a json file as storage for the Manifest
type JSONProvider struct {
}

func (jp *JSONProvider) Read(path string) (Manifest, error) {
	var res Manifest
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return Manifest{}, err
	}

	return res, nil
}

func (jp *JSONProvider) Write(path string, manifest Manifest) error {
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, b, 0640)
}
