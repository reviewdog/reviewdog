package reviewdog

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/reviewdog/errorformat"

	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/parser"
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

func TestReviewdog_Run_logs_filter_summary(t *testing.T) {
	defaultLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })

	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))

	c := &testWriter{
		FakePost: func(*Comment) error {
			t.Fatal("filtered diagnostic should not be posted")
			return nil
		},
	}
	efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`})
	app := NewReviewdog(
		"tool name",
		parser.NewErrorformatParser(efm),
		c,
		NewDiffString("", 1),
		filter.ModeAdded,
		FailLevelDefault,
	)

	if err := app.Run(context.Background(), strings.NewReader("outside.go:1:1: message\n")); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`msg="reviewdog: filter summary"`,
		`tool="tool name"`,
		`filter-mode=added`,
		`diagnostics=1`,
		`reportable=0`,
		`filtered=1`,
	} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("debug log %q does not contain %q", logs.String(), want)
		}
	}
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
