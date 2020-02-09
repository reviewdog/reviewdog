package reviewdog

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/reviewdog/reviewdog/diff"
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
		"added"}

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

// FilteredCheck represents CheckResult with filtering info.
type FilteredCheck struct {
	*CheckResult
	InDiff   bool
	LnumDiff int
}

// FilterCheck filters check results by diff. It doesn't drop check which
// is not in diff but set FilteredCheck.InDiff field false.
func FilterCheck(results []*CheckResult, diff []*diff.FileDiff, strip int, wd string, filterMode FilterMode) []*FilteredCheck {
	checks := make([]*FilteredCheck, 0, len(results))

	var filterFn lineComparator

	switch filterMode {
	case FilterModeDiffContext:
		filterFn = anyLine
	case FilterModeAdded:
		filterFn = isAddedLine
	}

	significantlines := significantDiffLines(diff, filterFn, strip)

	for _, result := range results {
		check := &FilteredCheck{CheckResult: result}

		significantline := significantlines.Get(result.Path, result.Lnum)
		result.Path = CleanPath(result.Path, wd)
		if significantline != nil {
			check.InDiff = true
			check.LnumDiff = significantline.LnumDiff
		}

		checks = append(checks, check)
	}

	return checks
}

// CleanPath clean up given path. If workdir is not empty, it returns relative
// path to the given workdir.
func CleanPath(path, workdir string) string {
	p := path
	if filepath.IsAbs(path) && workdir != "" {
		relPath, err := filepath.Rel(workdir, path)
		if err == nil {
			p = relPath
		}
	}
	p = filepath.Clean(p)
	if p == "." {
		return ""
	}
	return filepath.ToSlash(p)
}

// significantLine represents the line in diff we want to filter check results by.
type significantLine struct {
	Path     string // path to new file
	Lnum     int    // the line number in the new file
	LnumDiff int    // the line number of the diff (Same as Lnumdiff of diff.Line)
	Content  string // line content
}

// posToSignificantLine is a hash table of normalized path to line number to significantLine.
type posToSignificantLine map[string]map[int]*significantLine

func (p posToSignificantLine) Get(path string, lnum int) *significantLine {
	npath, err := normalizePath(path)
	if err != nil {
		return nil
	}
	ltodiff, ok := p[npath]
	if !ok {
		return nil
	}
	diffline, ok := ltodiff[lnum]
	if !ok {
		return nil
	}
	return diffline
}

type lineComparator func(diff.Line) bool

func isAddedLine(line diff.Line) bool {
	return line.Type == diff.LineAdded
}

func anyLine(line diff.Line) bool {
	return true
}

// significantDiffLines traverse []*diff.FileDiff and returns posToSignificantLine.
func significantDiffLines(filediffs []*diff.FileDiff, isSignificantLine lineComparator, strip int) posToSignificantLine {
	r := make(posToSignificantLine)
	for _, filediff := range filediffs {
		path := filediff.PathNew
		ltodiff := make(map[int]*significantLine)
		if strip > 0 {
			ps := strings.Split(filepath.ToSlash(filediff.PathNew), "/")
			if len(ps) > strip {
				path = filepath.Join(ps[strip:]...)
			}
		}
		np, err := normalizePath(path)
		if err != nil {
			log.Printf("reviewdog: failed to normalize path: %s", path)
			continue
		}
		path = np

		for _, hunk := range filediff.Hunks {
			for _, line := range hunk.Lines {
				if isSignificantLine(*line) {
					ltodiff[line.LnumNew] = &significantLine{
						Path:     path,
						Lnum:     line.LnumNew,
						LnumDiff: line.LnumDiff,
						Content:  line.Content,
					}
				}
			}
		}
		r[path] = ltodiff
	}
	return r
}

func normalizePath(p string) (string, error) {
	if !filepath.IsAbs(p) {
		path, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		p = path
	}
	return filepath.ToSlash(p), nil
}
