package filter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

// Mode represents enumeration of available filter modes
type Mode int

const (
	// ModeDefault represents default mode, which means users doesn't specify
	// filter-mode. The behavior can be changed depending on reporters/context
	// later if we want. Basically, it's same as ModeAdded because it's most safe
	// and basic mode for reporters implementation.
	ModeDefault Mode = iota
	// ModeAdded represents filtering by added/changed diff lines.
	ModeAdded
	// ModeDiffContext represents filtering by diff context.
	// i.e. changed lines +-N lines (e.g. N=3 for default git diff).
	ModeDiffContext
	// ModeFile represents filtering by changed files.
	ModeFile
	// ModeNoFilter doesn't filter out any results.
	ModeNoFilter
)

// String implements the flag.Value interface
func (mode *Mode) String() string {
	names := [...]string{
		"default",
		"added",
		"diff_context",
		"file",
		"nofilter",
	}
	if *mode < ModeDefault || *mode > ModeNoFilter {
		return "Unknown mode"
	}

	return names[*mode]
}

// Set implements the flag.Value interface
func (mode *Mode) Set(value string) error {
	switch value {
	case "default", "":
		*mode = ModeDefault
	case "added":
		*mode = ModeAdded
	case "diff_context":
		*mode = ModeDiffContext
	case "file":
		*mode = ModeFile
	case "nofilter":
		*mode = ModeNoFilter
	default:
		return fmt.Errorf("invalid mode name: %s", value)
	}
	return nil
}

// DiffFilter filters lines by diff.
type DiffFilter struct {
	// Current working directory (workdir).
	cwd string

	// Relative path to the project root (e.g. git) directory from current workdir.
	// It can be empty if it doesn't find any project root directory.
	projectRelPath string

	strip int
	mode  Mode

	difflines difflines
	difffiles difffiles
}

// difflines is a hash table of normalizedPath to line number to *diff.Line.
type difflines map[normalizedPath]map[int]*diff.Line

// difffiles is a hash table of normalizedPath to *diff.FileDiff.
type difffiles map[normalizedPath]*diff.FileDiff

// NewDiffFilter creates a new DiffFilter.
func NewDiffFilter(diff []*diff.FileDiff, strip int, cwd string, mode Mode) *DiffFilter {
	df := &DiffFilter{
		strip:     strip,
		cwd:       cwd,
		mode:      mode,
		difflines: make(difflines),
		difffiles: make(difffiles),
	}
	// If cwd is empty, projectRelPath should not have any meaningful data too.
	if cwd != "" {
		df.projectRelPath, _ = serviceutil.GitRelWorkdir()
	}
	df.addDiff(diff)
	return df
}

func (df *DiffFilter) addDiff(filediffs []*diff.FileDiff) {
	for _, filediff := range filediffs {
		path := df.normalizeDiffPath(filediff)
		df.difffiles[path] = filediff
		lines, ok := df.difflines[path]
		if !ok {
			lines = make(map[int]*diff.Line)
		}
		for _, hunk := range filediff.Hunks {
			for _, line := range hunk.Lines {
				if line.LnumNew > 0 {
					lines[line.LnumNew] = line
				}
			}
		}
		df.difflines[path] = lines
	}
}

// ShouldReport returns true, if the given path should be reported depending on
// the filter Mode. It also optionally return diff file/line.
func (df *DiffFilter) ShouldReport(path string, lnum int) (bool, *diff.FileDiff, *diff.Line) {
	npath := df.normalizePath(path)
	file := df.difffiles[npath]
	lines, ok := df.difflines[npath]
	if !ok {
		return df.mode == ModeNoFilter, file, nil
	}
	line, ok := lines[lnum]
	if !ok {
		return df.mode == ModeNoFilter || df.mode == ModeFile, file, nil
	}
	return df.isSignificantLine(line), file, line
}

// DiffLine returns diff data from given new path and lnum. Returns nil if not
// found.
func (df *DiffFilter) DiffLine(path string, lnum int) *diff.Line {
	npath := df.normalizePath(path)
	lines, ok := df.difflines[npath]
	if !ok {
		return nil
	}
	line, ok := lines[lnum]
	if !ok {
		return nil
	}
	return line
}

func (df *DiffFilter) isSignificantLine(line *diff.Line) bool {
	switch df.mode {
	case ModeDiffContext, ModeFile, ModeNoFilter:
		return true // any lines in diff are significant.
	case ModeAdded, ModeDefault:
		return line.Type == diff.LineAdded
	}
	return false
}

// normalizedPath is file path which is relative to **project root dir** or
// to current dir if project root not found.
type normalizedPath struct{ p string }

func (df *DiffFilter) normalizePath(path string) normalizedPath {
	return normalizedPath{p: NormalizePath(path, df.cwd, df.projectRelPath)}
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
	return normalizedPath{p: NormalizeDiffPath(filediff.PathNew, df.strip)}
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

func splitPathList(path string) []string {
	return strings.Split(filepath.ToSlash(path), "/")
}
