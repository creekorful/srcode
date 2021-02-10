package manifest

import (
	"errors"
	"strings"
)

var (
	// ErrNoProjectFound is returned when no project exist at given path
	ErrNoProjectFound = errors.New("no project exist at given path")
	// ErrScriptNotFound is returned when given script is not found
	ErrScriptNotFound = errors.New("no script with the name found")
)

// Manifest is the representation of the codebase
type Manifest struct {
	Projects map[string]Project  `json:"projects,omitempty"`
	Scripts  map[string][]string `json:"scripts,omitempty"`
}

// Project is a Codebase project
type Project struct {
	Remote  string              `json:"remote"`
	Config  map[string]string   `json:"config,omitempty"`
	Scripts map[string][]string `json:"scripts,omitempty"`
	Hook    string              `json:"hook,omitempty"`
}

// GetScript is an helper method to retrieve project script
func (m *Manifest) GetScript(projectPath, scriptName string) ([]string, error) {
	// Retrieve project
	project, exist := m.Projects[projectPath]
	if !exist {
		return nil, ErrNoProjectFound
	}

	// Check if script is defined locally
	scriptVal, exist := project.Scripts[scriptName]
	if !exist {
		return nil, ErrScriptNotFound
	}

	// It's a script alias
	if len(scriptVal) == 1 && strings.HasPrefix(scriptVal[0], "@") {
		scriptVal, exist = m.Scripts[strings.TrimPrefix(scriptVal[0], "@")]
		if !exist {
			return nil, ErrScriptNotFound
		}
	}

	return scriptVal, nil
}
