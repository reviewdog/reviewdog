package reviewdog

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/haya14busa/errorformat"
	"github.com/haya14busa/reviewdog/diff"
)

func ExampleReviewdog() {
	difftext := `diff --git a/golint.old.go b/golint.new.go
index 34cacb9..a727dd3 100644
--- a/golint.old.go
+++ b/golint.new.go
@@ -2,6 +2,12 @@ package test
 
 var V int
 
+var NewError1 int
+
 // invalid func comment
 func F() {
 }
+
+// invalid func comment2
+func F2() {
+}
`
	lintresult := `golint.new.go:3:5: exported var V should have comment or be unexported
golint.new.go:5:5: exported var NewError1 should have comment or be unexported
golint.new.go:7:1: comment on exported function F should be of the form "F ..."
golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
`
	efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`})
	p := NewErrorformatParser(efm)
	c := NewCommentWriter(os.Stdout)
	d := NewDiffString(difftext, 1)
	app := NewReviewdog("tool name", p, c, d)
	app.Run(strings.NewReader(lintresult))
	// Unordered output:
	// golint.new.go:5:5: exported var NewError1 should have comment or be unexported
	// golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
}

func TestAddedDiffLines(t *testing.T) {
	content := `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
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

	filediffs, _ := diff.ParseMultiFile(strings.NewReader(content))
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
