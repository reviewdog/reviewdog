package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var A1 = 14
var A2 = 14
var A3 = 14
var A4 = 14
var A5 = 14
var A6 = 14
var A7 = 14
var A8 = 14
var A9 = 14
var A10 = 14
var A11 = 14
var A12 = 14
var A13 = 14
var A14 = 14
var A15 = 14
var A16 = 14
var A17 = 14
var A18 = 14
var A19 = 14
var A20 = 14
var A21 = 14
var A22 = 14
var A23 = 14
var A24 = 14
var A25 = 14
var A26 = 14
var A27 = 14
var A28 = 14
var A29 = 14
var A30 = 14
var A31 = 14
var A32 = 14
var A33 = 14
var A34 = 14
var A35 = 14
var A36 = 14
var A37 = 14
var A38 = 14
var A39 = 14
var A40 = 14
var A41 = 14
var A42 = 14
var A43 = 14
var A44 = 14
var A45 = 14
var A46 = 14
var A47 = 14
var A48 = 14
var A49 = 14
var A50 = 14
var A51 = 14
var A52 = 14
var A53 = 14
var A54 = 14
var A55 = 14
var A56 = 14
var A57 = 14
var A58 = 14
var A59 = 14
var A60 = 14
var A61 = 14
var A62 = 14
var A63 = 14
var A64 = 14
var A65 = 14
var A66 = 14
var A67 = 14
var A68 = 14
var A69 = 14
var A70 = 14
var A71 = 14
var A72 = 14
var A73 = 14
var A74 = 14
var A75 = 14
var A76 = 14
var A77 = 14
var A78 = 14
var A79 = 14
var A80 = 14
var A81 = 14
var A82 = 14
var A83 = 14
var A84 = 14
var A85 = 14
var A86 = 14
var A87 = 14
var A88 = 14
var A89 = 14
var A90 = 14
var A91 = 14
var A92 = 14
var A93 = 14
var A94 = 14
var A95 = 14
var A96 = 14
var A97 = 14
var A98 = 14
var A99 = 14
var A100 = 14

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
