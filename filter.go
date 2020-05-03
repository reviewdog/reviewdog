package reviewdog

import (
	"path/filepath"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/difffilter"
)

// FilteredCheck represents CheckResult with filtering info.
type FilteredCheck struct {
	*CheckResult
	InDiff   bool
	LnumDiff int
}

// FilterCheck filters check results by diff. It doesn't drop check which
// is not in diff but set FilteredCheck.InDiff field false.
func FilterCheck(results []*CheckResult, diff []*diff.FileDiff, strip int,
	cwd string, mode difffilter.FilterMode) []*FilteredCheck {
	checks := make([]*FilteredCheck, 0, len(results))
	df := difffilter.New(diff, strip, cwd, mode)
	for _, result := range results {
		check := &FilteredCheck{CheckResult: result}
		if yes, lnumdiff := df.InDiff(result.Path, result.Lnum); yes {
			check.InDiff = true
			check.LnumDiff = lnumdiff
		}
		result.Path = CleanPath(result.Path, cwd)
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
