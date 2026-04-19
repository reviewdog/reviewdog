package reviewdog

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/reviewdog/errorformat"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/parser"
	"github.com/reviewdog/reviewdog/pathutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ CommentService = &testWriter{}

type testWriter struct {
	FakePost               func(c *Comment) error
	shouldPrependGitRelDir bool
}

func (s *testWriter) Post(_ context.Context, c *Comment) error {
	return s.FakePost(c)
}

func (s *testWriter) ShouldPrependGitRelDir() bool { return s.shouldPrependGitRelDir }

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
	p := parser.NewErrorformatParser(efm)
	c := NewRawCommentWriter(os.Stdout)
	d := NewDiffString(difftext, 1)
	app := NewReviewdog("tool name", p, c, d, filter.ModeAdded, FailLevelDefault)
	app.Run(context.Background(), strings.NewReader(lintresult))
	// Unordered output:
	// golint.new.go:5:5: exported var NewError1 should have comment or be unexported
	// golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
}

func TestReviewdog_Run_clean_path(t *testing.T) {
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
	lintresult := `./golint.new.go:3:5: exported var V should have comment or be unexported
./golint.new.go:5:5: exported var NewError1 should have comment or be unexported
./golint.new.go:7:1: comment on exported function F should be of the form "F ..."
./golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
`

	want := "golint.new.go"

	c := &testWriter{
		FakePost: func(c *Comment) error {
			if got := c.Result.Diagnostic.GetLocation().GetPath(); got != want {
				t.Errorf("path: got %v, want %v", got, want)
			}
			return nil
		},
	}

	efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`})
	p := parser.NewErrorformatParser(efm)
	d := NewDiffString(difftext, 1)
	app := NewReviewdog("tool name", p, c, d, filter.ModeAdded, FailLevelDefault)
	app.Run(context.Background(), strings.NewReader(lintresult))
}

func TestReviewdog_Run_git_rel_dir(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("./_testdata/")

	difftext := `diff --git a/golint.old.go b/golint.new.go
index 34cacb9..a727dd3 100644
--- a/_testdata/golint.old.go
+++ b/_testdata/golint.new.go
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
	lintresult := `./golint.new.go:3:5: exported var V should have comment or be unexported
./golint.new.go:5:5: exported var NewError1 should have comment or be unexported
./golint.new.go:7:1: comment on exported function F should be of the form "F ..."
./golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
`

	want := "_testdata/golint.new.go"

	c := &testWriter{
		FakePost: func(c *Comment) error {
			if got := c.Result.Diagnostic.GetLocation().GetPath(); got != want {
				t.Errorf("path: got %v, want %v", got, want)
			}
			return nil
		},
		shouldPrependGitRelDir: true,
	}

	efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`})
	p := parser.NewErrorformatParser(efm)
	d := NewDiffString(difftext, 1)
	app := NewReviewdog("tool name", p, c, d, filter.ModeAdded, FailLevelDefault)
	app.Run(context.Background(), strings.NewReader(lintresult))
}

func TestReviewdog_Run_git_rel_dir_sarif(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir("./_testdata/")

	content, err := os.ReadFile("golint.go")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile("golint.new.go", content, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove("golint.new.go")
	})

	difftext := strings.Join([]string{
		"diff --git a/golint.old.go b/golint.new.go",
		"index 34cacb9..a727dd3 100644",
		"--- a/_testdata/golint.old.go",
		"+++ b/_testdata/golint.new.go",
		"@@ -2,6 +2,12 @@ package test",
		" ",
		" var V int",
		" ",
		"+var NewError1 int",
		"+",
		" // invalid func comment",
		" func F() {",
		" }",
		"+",
		"+// invalid func comment2",
		"+func F2() {",
		"+}",
	}, "\n")
	sarifResult := `{
  "runs": [
    {
      "results": [
        {
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "_testdata/golint.new.go"
                },
                "region": {
                  "startLine": 5,
                  "startColumn": 5
                }
              }
            }
          ],
          "message": {
            "text": "exported var NewError1 should have comment or be unexported"
          }
        }
      ],
      "tool": {
        "driver": {
          "name": "sarif-tool"
        }
      }
    }
  ]
}`

	results, err := parser.NewSarifParser().Parse(strings.NewReader(sarifResult))
	if err != nil {
		t.Fatalf("parse sarif: %v", err)
	}
	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	gitRelWorkdir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		t.Fatalf("git rel workdir: %v", err)
	}
	pathutil.NormalizePathInResults(results, workdir, gitRelWorkdir)
	if got := results[0].GetLocation().GetPath(); got != "_testdata/golint.new.go" {
		t.Fatalf("path: got %v, want %v", got, "_testdata/golint.new.go")
	}

	filediffs, err := diff.ParseMultiFile(bytes.NewReader([]byte(difftext)))
	if err != nil {
		t.Fatalf("parse diff: %v", err)
	}
	checks := filter.FilterCheck(results, filediffs, 1, workdir, filter.ModeAdded)
	if len(checks) != 1 {
		t.Fatalf("got %d checks, want 1", len(checks))
	}
	if !checks[0].ShouldReport {
		t.Fatal("expected SARIF diagnostic to be reported in subdirectory context")
	}
}

func TestReviewdog_Run_returns_nil_if_fail_on_error_not_passed_and_some_errors_found(t *testing.T) {
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

	c := NewRawCommentWriter(os.Stdout)
	efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`})
	p := parser.NewErrorformatParser(efm)
	d := NewDiffString(difftext, 1)
	app := NewReviewdog("tool name", p, c, d, filter.ModeAdded, FailLevelDefault)
	err := app.Run(context.Background(), strings.NewReader(lintresult))

	if err != nil {
		t.Errorf("No errors expected, but got %v", err)
	}
}

func TestReviewdog_Run_returns_error_if_fail_on_error_passed_and_some_errors_found(t *testing.T) {
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
	c := NewRawCommentWriter(os.Stdout)
	efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`})
	p := parser.NewErrorformatParser(efm)
	d := NewDiffString(difftext, 1)
	app := NewReviewdog("tool name", p, c, d, filter.ModeAdded, FailLevelAny)
	err := app.Run(context.Background(), strings.NewReader(lintresult))

	if err != nil && err.Error() != "input data has violations" {
		t.Errorf("'input data has violations' expected, but got %v", err)
	}
}
