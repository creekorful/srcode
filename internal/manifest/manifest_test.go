package manifest

import (
	"reflect"
	"testing"
)

func TestManifest_GetScript(t *testing.T) {
	m := Manifest{
		Projects: map[string]Project{
			"project-1": {Scripts: map[string][]string{"test": {"test-local"}}},
			"project-2": {Scripts: map[string][]string{"test": {"@test-global"}}},
		},
		Scripts: map[string][]string{"test-global": {"test-global-42"}},
	}

	if _, err := m.GetScript("test", "test"); err != ErrNoProjectFound {
		t.Fail()
	}

	if _, err := m.GetScript("project-1", "test-12"); err != ErrScriptNotFound {
		t.Fail()
	}

	if val, err := m.GetScript("project-1", "test"); err != nil || !reflect.DeepEqual(val, []string{"test-local"}) {
		t.Fail()
	}

	if val, err := m.GetScript("project-2", "test"); err != nil || !reflect.DeepEqual(val, []string{"test-global-42"}) {
		t.Fail()
	}
}
