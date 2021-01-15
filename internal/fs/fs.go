package fs

import (
	"fmt"
	"strings"
)

// GetParentDirs return the parent directories of given path
func GetParentDirs(path string) []string {
	parts := strings.Split(path, "/")
	var dirs []string

	c := ""
	for _, part := range parts {
		if part != "" {
			c = fmt.Sprintf("%s/%s", c, part)
			dirs = append(dirs, c)
		}
	}

	// Remove last dir (because children)
	dirs = dirs[:len(dirs)-1]

	return dirs
}
