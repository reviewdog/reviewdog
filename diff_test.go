package reviewdog

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func TestDiffString(t *testing.T) {
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
	d := NewDiffString(difftext, 1)
	b, err := d.Diff(context.Background())
	if err != nil {
		t.Error(err)
	}
	got := string(b)
	if got != difftext {
		t.Errorf("got:\n%v\nwant:\n%v", got, difftext)
	}
}

func TestDiffCmd(t *testing.T) {
	wantb, err := os.ReadFile("./diff/testdata/golint.diff")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.SplitN(string(wantb), "\n", 5)[4] // strip extended header
	cmd := exec.Command("git", "diff", "--no-index", "./diff/testdata/golint.old.go", "./diff/testdata/golint.new.go")
	d := NewDiffCmd(cmd, 1)
	// ensure it supports multiple use
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := d.Diff(context.Background())
			if err != nil {
				t.Error(string(b), err)
			}
			got := strings.SplitN(string(b), "\n", 5)[4]
			if got != want {
				t.Errorf("got:\n%v\nwant:\n%v", got, want)
			}
		}()
	}
	wg.Wait()
}
