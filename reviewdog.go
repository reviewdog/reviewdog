package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var Y1 = 14
var Y2 = 14
var Y3 = 14
var Y4 = 14
var Y5 = 14
var Y6 = 14
var Y7 = 14
var Y8 = 14
var Y9 = 14
var Y10 = 14
var Y11 = 14
var Y12 = 14
var Y13 = 14
var Y14 = 14
var Y15 = 14
var Y16 = 14
var Y17 = 14
var Y18 = 14
var Y19 = 14
var Y20 = 14
var Y21 = 14
var Y22 = 14
var Y23 = 14
var Y24 = 14
var Y25 = 14
var Y26 = 14
var Y27 = 14
var Y28 = 14
var Y29 = 14
var Y30 = 14
var Y31 = 14

// var Y32 = 14
// var Y33 = 14
// var Y34 = 14
// var Y35 = 14
// var Y36 = 14
// var Y37 = 14
// var Y38 = 14
// var Y39 = 14
// var Y40 = 14
// var Y41 = 14
// var Y42 = 14
// var Y43 = 14
// var Y44 = 14
// var Y45 = 14
// var Y46 = 14
// var Y47 = 14
// var Y48 = 14
// var Y49 = 14
// var Y50 = 14

// var Y51 = 14
// var Y52 = 14
// var Y53 = 14
// var Y54 = 14
// var Y55 = 14
// var Y56 = 14
// var Y57 = 14
// var Y58 = 14
// var Y59 = 14
// var Y60 = 14
// var Y61 = 14
// var Y62 = 14
// var Y63 = 14
// var Y64 = 14
// var Y65 = 14
// var Y66 = 14
// var Y67 = 14
// var Y68 = 14
// var Y69 = 14
// var Y70 = 14
// var Y71 = 14
// var Y72 = 14
// var Y73 = 14
// var Y74 = 14
// var Y75 = 14
// var Y76 = 14
// var Y77 = 14
// var Y78 = 14
// var Y79 = 14
// var Y80 = 14
// var Y81 = 14
// var Y82 = 14
// var Y83 = 14
// var Y84 = 14
// var Y85 = 14
// var Y86 = 14
// var Y87 = 14
// var Y88 = 14
// var Y89 = 14
// var Y90 = 14
// var Y91 = 14
// var Y92 = 14
// var Y93 = 14
// var Y94 = 14
// var Y95 = 14
// var Y96 = 14
// var Y97 = 14
// var Y98 = 14
// var Y99 = 14
// var Y100 = 14

// Reviewdog represents review dog application which parses result of compiler
// or linter, get diff and filter the results by diff, and report filtered
// results.
type Reviewdog struct {
	toolname string
	p        Parser
	c        CommentService
	d        DiffService
}

// NewReviewdog returns a new Reviewdog.
func NewReviewdog(toolname string, p Parser, c CommentService, d DiffService) *Reviewdog {
	return &Reviewdog{p: p, c: c, d: d, toolname: toolname}
}

func RunFromResult(ctx context.Context, c CommentService, results []*CheckResult,
	filediffs []*diff.FileDiff, strip int, toolname string) error {
	return (&Reviewdog{c: c, toolname: toolname}).runFromResult(ctx, results, filediffs, strip)
}

// CheckResult represents a checked result of static analysis tools.
// :h error-file-format
type CheckResult struct {
	Path    string   // relative file path
	Lnum    int      // line number
	Col     int      // column number (1 <tab> == 1 character column)
	Message string   // error message
	Lines   []string // Original error lines (often one line)
}

// Parser is an interface which parses compilers, linters, or any tools
// results.
type Parser interface {
	Parse(r io.Reader) ([]*CheckResult, error)
}

// Comment represents a reported result as a comment.
type Comment struct {
	*CheckResult
	Body     string
	LnumDiff int
	ToolName string
}

// CommentService is an interface which posts Comment.
type CommentService interface {
	Post(context.Context, *Comment) error
}

// BulkCommentService posts comments all at once when Flush() is called.
// Flush() will be called at the end of reviewdog run.
type BulkCommentService interface {
	CommentService
	Flush(context.Context) error
}

// DiffService is an interface which get diff.
type DiffService interface {
	Diff(context.Context) ([]byte, error)
	Strip() int
}

func (w *Reviewdog) runFromResult(ctx context.Context, results []*CheckResult,
	filediffs []*diff.FileDiff, strip int) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	checks := FilterCheck(results, filediffs, strip, wd)
	for _, check := range checks {
		if !check.InDiff {
			continue
		}
		comment := &Comment{
			CheckResult: check.CheckResult,
			Body:        check.Message, // TODO: format message
			LnumDiff:    check.LnumDiff,
			ToolName:    w.toolname,
		}
		if err := w.c.Post(ctx, comment); err != nil {
			return err
		}
	}

	if bulk, ok := w.c.(BulkCommentService); ok {
		return bulk.Flush(ctx)
	}

	return nil
}

// Run runs Reviewdog application.
func (w *Reviewdog) Run(ctx context.Context, r io.Reader) error {
	results, err := w.p.Parse(r)
	if err != nil {
		return fmt.Errorf("parse error: %v", err)
	}

	d, err := w.d.Diff(ctx)
	if err != nil {
		return fmt.Errorf("fail to get diff: %v", err)
	}

	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("fail to parse diff: %v", err)
	}

	return w.runFromResult(ctx, results, filediffs, w.d.Strip())
}
