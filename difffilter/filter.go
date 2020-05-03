package difffilter

import (
	"path/filepath"
	"strings"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

// FilterMode represents enumeration of available filter modes
type FilterMode int

const (
	// FilterModeDiffContext represents filtering by diff context
	FilterModeDiffContext FilterMode = iota
	// FilterModeAdded represents filtering by added diff lines
	FilterModeAdded
)

// String implements the flag.Value interface
func (mode *FilterMode) String() string {
	names := [...]string{
		"diff_context",
		"added",
	}
	if *mode < FilterModeDiffContext || *mode > FilterModeAdded {
		return "Unknown"
	}

	return names[*mode]
}

// Set implements the flag.Value interface
func (mode *FilterMode) Set(value string) error {
	switch value {
	case "diff_context":
		*mode = FilterModeDiffContext
	case "added":
		*mode = FilterModeAdded
	default:
		*mode = FilterModeDiffContext
	}
	return nil
}

// DiffFilter filters lines by diff.
type DiffFilter struct {
	// Current working directory (workdir).
	cwd string

	// Relative path to the project root (e.g. git) directory from current workdir.
	// It can be empty if it doesn't find any project root directory.
	relPathToProjectRoot string

	strip int
	mode  FilterMode

	difflines difflines
}

// difflines is a hash table of normalizedPath to line number to diff.Line.
type difflines map[normalizedPath]map[int]*diff.Line

// New creates a new DiffFilter.
func New(diff []*diff.FileDiff, strip int, cwd string, mode FilterMode) *DiffFilter {
	df := &DiffFilter{
		strip:     strip,
		cwd:       cwd,
		mode:      mode,
		difflines: make(difflines),
	}
	df.relPathToProjectRoot, _ = serviceutil.GitRelWorkdir()
	df.addDiff(diff)
	return df
}

func (df *DiffFilter) addDiff(filediffs []*diff.FileDiff) {
	for _, filediff := range filediffs {
		path := df.normalizeDiffPath(filediff)
		lines, ok := df.difflines[path]
		if !ok {
			lines = make(map[int]*diff.Line)
		}
		for _, hunk := range filediff.Hunks {
			for _, line := range hunk.Lines {
				if df.isSignificantLine(line) {
					lines[line.LnumNew] = line
				}
			}
		}
		df.difflines[path] = lines
	}
}

// InDiff returns true, if the given path is in diff. It also optinally return
// LnumDiff[1].
//
// [1]: https://github.com/reviewdog/reviewdog/blob/73c40e69d937033b2cf20f2d6085fb7ef202e770/diff/diff.go#L81-L88
func (df *DiffFilter) InDiff(path string, lnum int) (yes bool, lnumdiff int) {
	lines, ok := df.difflines[df.normalizePath(path)]
	if !ok {
		return false, 0
	}
	line, ok := lines[lnum]
	if !ok {
		return false, 0
	}
	return true, line.LnumDiff
}

func (df *DiffFilter) isSignificantLine(line *diff.Line) bool {
	switch df.mode {
	case FilterModeDiffContext:
		return true // any lines in diff are significant.
	case FilterModeAdded:
		return line.Type == diff.LineAdded
	}
	return false
}

// normalizedPath is file path which is relative to **project root dir** or
// to current dir if project root not found.
type normalizedPath struct{ p string }

func (df *DiffFilter) normalizePath(path string) normalizedPath {
	path = filepath.Clean(path)
	// Convert absolute path to relative path only if the path is in current
	// directory.
	if filepath.IsAbs(path) && df.cwd != "" && contains(path, df.cwd) {
		relPath, err := filepath.Rel(df.cwd, path)
		if err == nil {
			path = relPath
		}
	}
	if !filepath.IsAbs(path) && df.relPathToProjectRoot != "" {
		path = filepath.Join(df.relPathToProjectRoot, path)
	}
	return normalizedPath{p: filepath.ToSlash(path)}
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

// Assuming diff path should be relative path to the project root dir by
// default (e.g. git diff).
//
// `git diff --relative` can returns relative path to current workdir, so we
// ask users not to use it for reviewdog command.
func (df *DiffFilter) normalizeDiffPath(filediff *diff.FileDiff) normalizedPath {
	path := filediff.PathNew
	if df.strip > 0 {
		ps := splitPathList(filediff.PathNew)
		if len(ps) > df.strip {
			path = filepath.Join(ps[df.strip:]...)
		}
	}
	return normalizedPath{p: filepath.ToSlash(filepath.Clean(path))}
}

func splitPathList(path string) []string {
	return strings.Split(filepath.ToSlash(path), "/")
}
