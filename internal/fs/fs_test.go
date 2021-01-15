package fs

import "testing"

func TestGetParentDirs(t *testing.T) {
	dirs := GetParentDirs("/a/b/c/d")

	if !contains(dirs, "/a/b/c") {
		t.Fail()
	}
	if !contains(dirs, "/a/b") {
		t.Fail()
	}
	if !contains(dirs, "/a") {
		t.Fail()
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
