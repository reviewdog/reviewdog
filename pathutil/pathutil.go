package pathutil

import (
	"path/filepath"
	"strings"
)

// NormalizePath return normalized path with workdir and relative path to
// project.
func NormalizePath(path, workdir, projectRelPath string) string {
	path = filepath.Clean(path)
	if path == "." {
		return ""
	}
	// Convert absolute path to relative path only if the path is in current
	// directory.
	if filepath.IsAbs(path) && workdir != "" && contains(path, workdir) {
		relPath, err := filepath.Rel(workdir, path)
		if err == nil {
			path = relPath
		}
	}
	if !filepath.IsAbs(path) && projectRelPath != "" {
		path = filepath.Join(projectRelPath, path)
	}
	return filepath.ToSlash(path)
}

// NormalizeDiffPath return path normalized path from given path in diff with
// strip.
func NormalizeDiffPath(diffpath string, strip int) string {
	if diffpath == "/dev/null" {
		return ""
	}
	path := diffpath
	if strip > 0 && !filepath.IsAbs(path) {
		ps := splitPathList(path)
		if len(ps) > strip {
			path = filepath.Join(ps[strip:]...)
		}
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func contains(path, base string) bool {
	ps := splitPathList(path)
	bs := splitPathList(base)
	if len(ps) < len(bs) {
		return false
	}
	for i := range bs {
		if bs[i] != ps[i] {
			return false
		}
	}
	return true
}

func splitPathList(path string) []string {
	return strings.Split(filepath.ToSlash(path), "/")
}
