package reviewdog

import (
	"path/filepath"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/difffilter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

// FilteredCheck represents Diagnostic with filtering info.
type FilteredCheck struct {
	Diagnostic   *rdf.Diagnostic
	ShouldReport bool
	// false if the result is outside diff files.
	InDiffFile bool
	// true if the result is inside a diff hunk.
	// If it's a multiline result, both start and end must be in the same diff
	// hunk.
	InDiffContext bool
	OldPath       string
	OldLine       int
}

// FilterCheck filters check results by diff. It doesn't drop check which
// is not in diff but set FilteredCheck.ShouldReport field false.
func FilterCheck(results []*rdf.Diagnostic, diff []*diff.FileDiff, strip int,
	cwd string, mode difffilter.Mode) []*FilteredCheck {
	checks := make([]*FilteredCheck, 0, len(results))
	df := difffilter.New(diff, strip, cwd, mode)
	for _, result := range results {
		check := &FilteredCheck{Diagnostic: result}
		loc := result.GetLocation()
		startLine := int(loc.GetRange().GetStart().GetLine())
		endLine := int(loc.GetRange().GetEnd().GetLine())
		if endLine == 0 {
			endLine = startLine
		}
		check.InDiffContext = true
		for l := startLine; l <= endLine; l++ {
			shouldReport, difffile, diffline := df.ShouldReport(loc.GetPath(), l)
			check.ShouldReport = check.ShouldReport || shouldReport
			// all lines must be in diff.
			check.InDiffContext = check.InDiffContext && diffline != nil
			if difffile != nil {
				check.InDiffFile = true
				if l == startLine {
					// TODO(haya14busa): Support endline as well especially for GitLab.
					check.OldPath, check.OldLine = getOldPosition(difffile, strip, loc.GetPath(), l)
				}
			}
		}
		loc.Path = CleanPath(loc.GetPath(), cwd)
		checks = append(checks, check)
	}
	return checks
}

// CleanPath clean up given path. If workdir is not empty, it returns relative
// path to the given workdir.
//
// TODO(haya14busa): DRY. Create shared logic between this and
// difffilter.normalizePath.
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

func getOldPosition(filediff *diff.FileDiff, strip int, newPath string, newLine int) (oldPath string, oldLine int) {
	if filediff == nil {
		return "", 0
	}
	if difffilter.NormalizeDiffPath(filediff.PathNew, strip) != newPath {
		return "", 0
	}
	oldPath = difffilter.NormalizeDiffPath(filediff.PathOld, strip)
	delta := 0
	for _, hunk := range filediff.Hunks {
		if newLine < hunk.StartLineNew {
			break
		}
		delta += hunk.LineLengthOld - hunk.LineLengthNew
		for _, line := range hunk.Lines {
			if line.LnumNew == newLine {
				return oldPath, line.LnumOld
			}
		}
	}
	return oldPath, newLine + delta
}
