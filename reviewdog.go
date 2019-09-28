package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var X1 = 14
var X2 = 14
var X3 = 14
var X4 = 14
var X5 = 14
var X6 = 14
var X7 = 14
var X8 = 14
var X9 = 14
var X10 = 14
var X11 = 14
var X12 = 14
var X13 = 14
var X14 = 14
var X15 = 14
var X16 = 14
var X17 = 14
var X18 = 14
var X19 = 14
var X20 = 14
var X21 = 14
var X22 = 14
var X23 = 14
var X24 = 14
var X25 = 14
var X26 = 14
var X27 = 14
var X28 = 14
var X29 = 14
var X30 = 14
var X31 = 14

// var X32 = 14
// var X33 = 14
// var X34 = 14
// var X35 = 14
// var X36 = 14
// var X37 = 14
// var X38 = 14
// var X39 = 14
// var X40 = 14
// var X41 = 14
// var X42 = 14
// var X43 = 14
// var X44 = 14
// var X45 = 14
// var X46 = 14
// var X47 = 14
// var X48 = 14
// var X49 = 14
// var X50 = 14

// var X51 = 14
// var X52 = 14
// var X53 = 14
// var X54 = 14
// var X55 = 14
// var X56 = 14
// var X57 = 14
// var X58 = 14
// var X59 = 14
// var X60 = 14
// var X61 = 14
// var X62 = 14
// var X63 = 14
// var X64 = 14
// var X65 = 14
// var X66 = 14
// var X67 = 14
// var X68 = 14
// var X69 = 14
// var X70 = 14
// var X71 = 14
// var X72 = 14
// var X73 = 14
// var X74 = 14
// var X75 = 14
// var X76 = 14
// var X77 = 14
// var X78 = 14
// var X79 = 14
// var X80 = 14
// var X81 = 14
// var X82 = 14
// var X83 = 14
// var X84 = 14
// var X85 = 14
// var X86 = 14
// var X87 = 14
// var X88 = 14
// var X89 = 14
// var X90 = 14
// var X91 = 14
// var X92 = 14
// var X93 = 14
// var X94 = 14
// var X95 = 14
// var X96 = 14
// var X97 = 14
// var X98 = 14
// var X99 = 14
// var X100 = 14

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
