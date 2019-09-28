package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var Z1 = 14
var Z2 = 14
var Z3 = 14
var Z4 = 14
var Z5 = 14
var Z6 = 14
var Z7 = 14
var Z8 = 14
var Z9 = 14
var Z10 = 14
var Z11 = 14
var Z12 = 14
var Z13 = 14
var Z14 = 14
var Z15 = 14
var Z16 = 14
var Z17 = 14
var Z18 = 14
var Z19 = 14
var Z20 = 14
var Z21 = 14
var Z22 = 14
var Z23 = 14
var Z24 = 14
var Z25 = 14
var Z26 = 14
var Z27 = 14
var Z28 = 14
var Z29 = 14
var Z30 = 14
var Z31 = 14
var Z32 = 14
var Z33 = 14
var Z34 = 14
var Z35 = 14
var Z36 = 14
var Z37 = 14
var Z38 = 14
var Z39 = 14
var Z40 = 14
var Z41 = 14
var Z42 = 14
var Z43 = 14
var Z44 = 14
var Z45 = 14
var Z46 = 14
var Z47 = 14
var Z48 = 14
var Z49 = 14

// var Z50 = 14
// var Z51 = 14
// var Z52 = 14
// var Z53 = 14
// var Z54 = 14
// var Z55 = 14
// var Z56 = 14
// var Z57 = 14
// var Z58 = 14
// var Z59 = 14
// var Z60 = 14
// var Z61 = 14
// var Z62 = 14
// var Z63 = 14
// var Z64 = 14
// var Z65 = 14
// var Z66 = 14
// var Z67 = 14
// var Z68 = 14
// var Z69 = 14
// var Z70 = 14
// var Z71 = 14
// var Z72 = 14
// var Z73 = 14
// var Z74 = 14
// var Z75 = 14
// var Z76 = 14
// var Z77 = 14
// var Z78 = 14
// var Z79 = 14
// var Z80 = 14
// var Z81 = 14
// var Z82 = 14
// var Z83 = 14
// var Z84 = 14
// var Z85 = 14
// var Z86 = 14
// var Z87 = 14
// var Z88 = 14
// var Z89 = 14
// var Z90 = 14
// var Z91 = 14
// var Z92 = 14
// var Z93 = 14
// var Z94 = 14
// var Z95 = 14
// var Z96 = 14
// var Z97 = 14
// var Z98 = 14
// var Z99 = 14
// var Z100 = 14

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
