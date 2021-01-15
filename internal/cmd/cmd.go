package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func ExecWithOutput(cmd *exec.Cmd) (string, error) {
	// capture stderr
	stdErr := bytes.NewBufferString("")
	cmd.Stderr = stdErr

	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error while running `%s`: %s", cmd.String(), stdErr)
	}

	return strings.TrimSuffix(string(b), "\n"), err
}
