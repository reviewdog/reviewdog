package serviceutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitRelWorkdir returns git relative workdir of current directory.
//
// It should return the same output as `git rev-parse --show-prefix`.
// It does not execute `git` command to avoid needless git binary dependency.
//
// An example problem due to `git` command dependency: https://github.com/reviewdog/reviewdog/issues/1158
func GitRelWorkdir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := findGitRoot(cwd)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(cwd, root) {
		return "", fmt.Errorf("cannot get GitRelWorkdir: cwd=%q, root=%q", cwd, root)
	}
	const separator = string(filepath.Separator)
	path := strings.Trim(strings.TrimPrefix(cwd, root), separator)
	if path != "" {
		path += separator
	}
	return path, nil
}

func GetGitRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findGitRoot(cwd)
}

func findGitRoot(path string) (string, error) {
	gitPath, err := findDotGitPath(path)
	if err != nil {
		return "", err
	}
	return filepath.Dir(gitPath), nil
}

func findDotGitPath(path string) (string, error) {
	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		fi, err := os.Stat(filepath.Join(path, ".git"))
		if err == nil {
			if !fi.IsDir() {
				return "", fmt.Errorf(".git exist but is not a directory")
			}
			return filepath.Join(path, ".git"), nil
		}
		if !os.IsNotExist(err) {
			// unknown error
			return "", err
		}

		// detect bare repo
		ok, err := isGitDir(path)
		if err != nil {
			return "", err
		}
		if ok {
			return path, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf(".git not found")
		}
		path = parent
	}
}

// ref: https://github.com/git/git/blob/3bab5d56259722843359702bc27111475437ad2a/setup.c#L328-L338
func isGitDir(path string) (bool, error) {
	markers := []string{"HEAD", "objects", "refs"}
	for _, marker := range markers {
		_, err := os.Stat(filepath.Join(path, marker))
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			// unknown error
			return false, err
		}
		return false, nil
	}
	return true, nil
}
