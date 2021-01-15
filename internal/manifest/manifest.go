package manifest

// Manifest is the representation of the codebase
type Manifest struct {
	Projects map[string]Project `json:"projects"`
	Commands map[string]string  `json:"commands"`
}

// Project is a Codebase project
type Project struct {
	Remote   string            `json:"remote"`
	Config   map[string]string `json:"config"`
	Commands map[string]string `json:"commands"`
}
