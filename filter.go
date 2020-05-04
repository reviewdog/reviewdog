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
	LnumDiff     int
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
		if yes, diffline := df.ShouldReport(result.Path, result.Lnum); yes {
			check.ShouldReport = true
			if diffline != nil {
				check.LnumDiff = diffline.LnumDiff
			}
		}
		result.Path = CleanPath(result.Path, cwd)
		check.OldPath, check.OldLine = getOldPosition(diff, strip, result.Path, result.Lnum)
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

func getOldPosition(filediffs []*diff.FileDiff, strip int, newPath string, newLine int) (oldPath string, oldLine int) {
	for _, filediff := range filediffs {
		if difffilter.NormalizeDiffPath(filediff.PathNew, strip) != newPath {
			continue
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
	return "", 0
}
