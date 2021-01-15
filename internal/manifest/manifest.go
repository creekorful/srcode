package manifest

type Manifest struct {
	Projects map[string]Project `json:"projects"`
	Commands map[string]string  `json:"commands"`
}

type Project struct {
	Remote   string            `json:"remote"`
	Config   map[string]string `json:"config"`
	Commands map[string]string `json:"commands"`
}
