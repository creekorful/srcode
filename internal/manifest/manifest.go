package manifest

// Manifest is the representation of the codebase
type Manifest struct {
	Projects map[string]Project  `json:"projects"`
	Scripts  map[string][]string `json:"scripts"`
}

// Project is a Codebase project
type Project struct {
	Remote  string              `json:"remote"`
	Config  map[string]string   `json:"config"`
	Scripts map[string][]string `json:"scripts"`
}
