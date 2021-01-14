package manifest

type Manifest struct {
	Projects map[string]Project `json:"projects"`
}

type Project struct {
	Remote string `json:"remote"`
}