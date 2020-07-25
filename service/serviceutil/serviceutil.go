package serviceutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitRelWorkdir returns git relative workdir of current directory.
func GitRelWorkdir() (string, error) {
	b, err := exec.Command("git", "rev-parse", "--show-prefix").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'git rev-parse --show-prefix': %w", err)
	}
	return strings.Trim(string(b), "\n"), nil
}
