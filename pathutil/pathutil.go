package pathutil

import (
	"path/filepath"
	"strings"

	"github.com/reviewdog/reviewdog/proto/rdf"
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

// NormalizePathInResults normalize file path in RDFormat results.
func NormalizePathInResults(results []*rdf.Diagnostic, cwd, gitRelWorkdir string) {
	for _, result := range results {
		normalizeLocation(result.GetLocation(), cwd, gitRelWorkdir)
		for _, rel := range result.GetRelatedLocations() {
			normalizeLocation(rel.GetLocation(), cwd, gitRelWorkdir)
		}
	}
}

func normalizeLocation(loc *rdf.Location, cwd, gitRelWorkdir string) {
	if loc != nil {
		loc.Path = NormalizePath(loc.GetPath(), cwd, gitRelWorkdir)
	}
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
