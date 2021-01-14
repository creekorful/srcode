package codebase

import (
	"github.com/creekorful/srcode/internal/manifest"
	"reflect"
	"testing"
)

func TestCodebase_Projects(t *testing.T) {
	projects := map[string]manifest.Project{
		"test/15": {Remote: "test"},
	}

	codebase := &codebase{
		manifest: manifest.Manifest{
			Projects: projects,
		},
	}

	if !reflect.DeepEqual(codebase.Projects(), projects) {
		t.FailNow()
	}
}
