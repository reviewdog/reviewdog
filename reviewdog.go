package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var B1 = 14
var B2 = 14
var B3 = 14
var B4 = 14
var B5 = 14
var B6 = 14
var B7 = 14
var B8 = 14
var B9 = 14
var B10 = 14
var B11 = 14
var B12 = 14
var B13 = 14
var B14 = 14
var B15 = 14
var B16 = 14
var B17 = 14
var B18 = 14
var B19 = 14
var B20 = 14
var B21 = 14
var B22 = 14
var B23 = 14
var B24 = 14
var B25 = 14
var B26 = 14
var B27 = 14
var B28 = 14
var B29 = 14
var B30 = 14
var B31 = 14
var B32 = 14
var B33 = 14
var B34 = 14
var B35 = 14
var B36 = 14
var B37 = 14
var B38 = 14
var B39 = 14
var B40 = 14
var B41 = 14
var B42 = 14
var B43 = 14
var B44 = 14
var B45 = 14
var B46 = 14
var B47 = 14
var B48 = 14
var B49 = 14
var B50 = 14
var B51 = 14
var B52 = 14
var B53 = 14
var B54 = 14
var B55 = 14
var B56 = 14
var B57 = 14
var B58 = 14
var B59 = 14
var B60 = 14
var B61 = 14
var B62 = 14
var B63 = 14
var B64 = 14
var B65 = 14
var B66 = 14
var B67 = 14
var B68 = 14
var B69 = 14
var B70 = 14
var B71 = 14
var B72 = 14
var B73 = 14
var B74 = 14
var B75 = 14
var B76 = 14
var B77 = 14
var B78 = 14
var B79 = 14
var B80 = 14
var B81 = 14
var B82 = 14
var B83 = 14
var B84 = 14
var B85 = 14
var B86 = 14
var B87 = 14
var B88 = 14
var B89 = 14
var B90 = 14
var B91 = 14
var B92 = 14
var B93 = 14
var B94 = 14
var B95 = 14
var B96 = 14
var B97 = 14
var B98 = 14
var B99 = 14
var B100 = 14

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
