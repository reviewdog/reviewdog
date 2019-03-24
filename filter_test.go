package reviewdog

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog/diff"
)

const diffContent = `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

func TestFilterCheck(t *testing.T) {
	results := []*CheckResult{
		{
			Path: "sample.new.txt",
			Lnum: 1,
		},
		{
			Path: "sample.new.txt",
			Lnum: 2,
		},
		{
			Path: "nonewline.new.txt",
			Lnum: 1,
		},
		{
			Path: "nonewline.new.txt",
			Lnum: 3,
		},
	}
	want := []*FilteredCheck{
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 1,
			},
			InDiff: false,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 2,
			},
			InDiff:   true,
			LnumDiff: 3,
		},
		{
			CheckResult: &CheckResult{
				Path: "nonewline.new.txt",
				Lnum: 1,
			},
			InDiff: false,
		},
		{
			CheckResult: &CheckResult{
				Path: "nonewline.new.txt",
				Lnum: 3,
			},
			InDiff:   true,
			LnumDiff: 5,
		},
	}
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	got := FilterCheck(results, filediffs, 0, "")
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestAddedDiffLines(t *testing.T) {
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	wd, _ := os.Getwd()
	wantlines := []string{
		"sample.new.txt:2:(difflnum:3) added line",
		"sample.new.txt:3:(difflnum:4) added line",
		"nonewline.new.txt:3:(difflnum:5) b",
		"nonewline.new.txt:4:(difflnum:6) b",
	}
	var gotlines []string
	for path, ltol := range addedDiffLines(filediffs, 0) {
		for lnum, addedline := range ltol {
			l := fmt.Sprintf("%v:%v:(difflnum:%v) %v", path[len(wd)+1:], lnum, addedline.LnumDiff, addedline.Content)
			gotlines = append(gotlines, l)
		}
	}
	sort.Strings(gotlines)
	sort.Strings(wantlines)
	if !reflect.DeepEqual(gotlines, wantlines) {
		t.Errorf("got:\n%v\nwant:\n%v", gotlines, wantlines)
	}
}
