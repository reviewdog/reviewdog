package watchdogs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/haya14busa/watchdogs/diff"
)

type golintParser struct{}

func (*golintParser) Parse(r io.Reader) ([]*CheckResult, error) {
	var rs []*CheckResult
	s := bufio.NewScanner(r)
	for s.Scan() {
		ps := strings.SplitN(s.Text(), ":", 4)
		r := &CheckResult{
			Path:    ps[0],
			Lnum:    mustAtoI(ps[1]),
			Col:     mustAtoI(ps[2]),
			Message: ps[3][1:],
		}
		rs = append(rs, r)
	}
	return rs, nil
}

func mustAtoI(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

type commentWriter struct {
	w io.Writer
}

func (w *commentWriter) Post(c *Comment) error {
	fmt.Fprintln(w.w, "---")
	fmt.Fprintf(w.w, "%v:%v:\n", c.Path, c.Lnum)
	fmt.Fprintln(w.w, c.Body)
	return nil
}

func ExampleWatchdogs() {
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
	p := &golintParser{}
	c := &commentWriter{w: os.Stdout}
	d := NewDiffString(difftext, 1)
	app := NewWatchdogs(p, c, d)
	app.Run(strings.NewReader(lintresult))
	// Output:
	// ---
	// golint.new.go:5:
	// exported var NewError1 should have comment or be unexported
	// ---
	// golint.new.go:11:
	// comment on exported function F2 should be of the form "F2 ..."
}

func ExampleAddedLines() {
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
	for path, ltol := range AddedLines(filediffs, 0) {
		for lnum, addedline := range ltol {
			fmt.Printf("%v:%v:(difflnum:%v) %v\n", path, lnum, addedline.LnumDiff, addedline.Content)
		}
	}
	// Output:
	// sample.new.txt:2:(difflnum:3) added line
	// sample.new.txt:3:(difflnum:4) added line
	// nonewline.new.txt:3:(difflnum:5) b
	// nonewline.new.txt:4:(difflnum:6) b
}
