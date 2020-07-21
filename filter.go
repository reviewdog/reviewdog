package reviewdog

import (
	"path/filepath"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/difffilter"
)

// FilteredCheck represents CheckResult with filtering info.
type FilteredCheck struct {
	*CheckResult
	ShouldReport bool
	LnumDiff     int  // 0 if the result is outside diff.
	InDiffFile   bool // false if the result is outside diff files.
	OldPath      string
	OldLine      int
}

// FilterCheck filters check results by diff. It doesn't drop check which
// is not in diff but set FilteredCheck.ShouldReport field false.
func FilterCheck(results []*CheckResult, diff []*diff.FileDiff, strip int,
	cwd string, mode difffilter.Mode) []*FilteredCheck {
	checks := make([]*FilteredCheck, 0, len(results))
	df := difffilter.New(diff, strip, cwd, mode)
	for _, result := range results {
		check := &FilteredCheck{CheckResult: result}
		loc := result.Diagnostic.GetLocation()
		lnum := int(loc.GetRange().GetStart().GetLine())
		shouldReport, difffile, diffline := df.ShouldReport(loc.GetPath(), lnum)
		check.ShouldReport = shouldReport
		if diffline != nil {
			check.LnumDiff = diffline.LnumDiff
		}
		loc.Path = CleanPath(loc.GetPath(), cwd)
		if difffile != nil {
			check.InDiffFile = true
			check.OldPath, check.OldLine = getOldPosition(difffile, strip, loc.GetPath(), lnum)
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
