package main

import "testing"

func TestParseGitConfig(t *testing.T) {
	config := parseGitConfig([]string{})
	if len(config) != 0 {
		t.Fail()
	}

	config = parseGitConfig([]string{"user.name=test", "user.email=something", "invalid"})
	if len(config) != 2 {
		t.Fail()
	}
	if config["user.name"] != "test" {
		t.Fail()
	}
	if config["user.email"] != "something" {
		t.Fail()
	}
}