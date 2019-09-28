package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var C1 = 14
var C2 = 14
var C3 = 14
var C4 = 14
var C5 = 14
var C6 = 14
var C7 = 14
var C8 = 14
var C9 = 14
var C10 = 14
var C11 = 14
var C12 = 14
var C13 = 14
var C14 = 14
var C15 = 14
var C16 = 14
var C17 = 14
var C18 = 14
var C19 = 14
var C20 = 14
var C21 = 14
var C22 = 14
var C23 = 14
var C24 = 14
var C25 = 14
var C26 = 14
var C27 = 14
var C28 = 14
var C29 = 14
var C30 = 14
var C31 = 14
var C32 = 14
var C33 = 14
var C34 = 14
var C35 = 14
var C36 = 14
var C37 = 14
var C38 = 14
var C39 = 14
var C40 = 14
var C41 = 14
var C42 = 14
var C43 = 14
var C44 = 14
var C45 = 14
var C46 = 14
var C47 = 14
var C48 = 14
var C49 = 14
var C50 = 14
var C51 = 14
var C52 = 14
var C53 = 14
var C54 = 14
var C55 = 14
var C56 = 14
var C57 = 14
var C58 = 14
var C59 = 14
var C60 = 14
var C61 = 14
var C62 = 14
var C63 = 14
var C64 = 14
var C65 = 14
var C66 = 14
var C67 = 14
var C68 = 14
var C69 = 14
var C70 = 14
var C71 = 14
var C72 = 14
var C73 = 14
var C74 = 14
var C75 = 14
var C76 = 14
var C77 = 14
var C78 = 14
var C79 = 14
var C80 = 14
var C81 = 14
var C82 = 14
var C83 = 14
var C84 = 14
var C85 = 14
var C86 = 14
var C87 = 14
var C88 = 14
var C89 = 14
var C90 = 14
var C91 = 14
var C92 = 14
var C93 = 14
var C94 = 14
var C95 = 14
var C96 = 14
var C97 = 14
var C98 = 14
var C99 = 14
var C100 = 14

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
