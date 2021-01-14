package manifest

import (
	"encoding/json"
	"io/ioutil"
)

//go:generate mockgen -destination=../manifest_mock/manifest_mock.go -package=manifest_mock . Provider

type Provider interface {
	Read(path string) (Manifest, error)
	Write(path string, manifest Manifest) error
}

type JsonProvider struct {
}

func (jp *JsonProvider) Read(path string) (Manifest, error) {
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

func (jp *JsonProvider) Write(path string, manifest Manifest) error {
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, b, 0640)
}
