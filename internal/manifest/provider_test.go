package manifest

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestJSONProvider_Read(t *testing.T) {
	m := Manifest{
		Projects: map[string]Project{
			"12": {Remote: "remote"},
		},
	}

	b, err := json.Marshal(m)
	if err != nil {
		t.FailNow()
	}

	path := filepath.Join(t.TempDir(), "test.json")
	if err := ioutil.WriteFile(path, b, 0640); err != nil {
		t.FailNow()
	}

	p := JSONProvider{}

	res, err := p.Read(path)
	if err != nil {
		t.FailNow()
	}

	if !reflect.DeepEqual(m, res) {
		t.Fail()
	}
}

func TestJSONProvider_Write(t *testing.T) {
	m := Manifest{
		Projects: map[string]Project{
			"12": {Remote: "remote"},
		},
	}

	p := JSONProvider{}

	path := filepath.Join(t.TempDir(), "test.json")
	if err := p.Write(path, m); err != nil {
		t.FailNow()
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.FailNow()
	}

	var res Manifest
	if err := json.Unmarshal(b, &res); err != nil {
		t.FailNow()
	}

	if !reflect.DeepEqual(m, res) {
		t.Fail()
	}
}
