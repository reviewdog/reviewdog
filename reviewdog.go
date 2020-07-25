package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/parser"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

// Reviewdog represents review dog application which parses result of compiler
// or linter, get diff and filter the results by diff, and report filtered
// results.
type Reviewdog struct {
	toolname    string
	p           parser.Parser
	c           CommentService
	d           DiffService
	filterMode  filter.Mode
	failOnError bool
}

// NewReviewdog returns a new Reviewdog.
func NewReviewdog(toolname string, p parser.Parser, c CommentService, d DiffService, filterMode filter.Mode, failOnError bool) *Reviewdog {
	return &Reviewdog{p: p, c: c, d: d, toolname: toolname, filterMode: filterMode, failOnError: failOnError}
}

// RunFromResult creates a new Reviewdog and runs it with check results.
func RunFromResult(ctx context.Context, c CommentService, results []*rdf.Diagnostic,
	filediffs []*diff.FileDiff, strip int, toolname string, filterMode filter.Mode, failOnError bool) error {
	return (&Reviewdog{c: c, toolname: toolname, filterMode: filterMode, failOnError: failOnError}).runFromResult(ctx, results, filediffs, strip, failOnError)
}

// Comment represents a reported result as a comment.
type Comment struct {
	Result   *filter.FilteredDiagnostic
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

func (w *Reviewdog) runFromResult(ctx context.Context, results []*rdf.Diagnostic,
	filediffs []*diff.FileDiff, strip int, failOnError bool) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	checks := filter.FilterCheck(results, filediffs, strip, wd, w.filterMode)
	hasViolations := false

	for _, check := range checks {
		if !check.ShouldReport {
			continue
		}
		comment := &Comment{
			Result:   check,
			ToolName: w.toolname,
		}
		if err := w.c.Post(ctx, comment); err != nil {
			return err
		}
		hasViolations = true
	}

	if bulk, ok := w.c.(BulkCommentService); ok {
		if err := bulk.Flush(ctx); err != nil {
			return err
		}
	}

	if failOnError && hasViolations {
		return fmt.Errorf("input data has violations")
	}

	return nil
}

// Run runs Reviewdog application.
func (w *Reviewdog) Run(ctx context.Context, r io.Reader) error {
	results, err := w.p.Parse(r)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	d, err := w.d.Diff(ctx)
	if err != nil {
		return fmt.Errorf("fail to get diff: %w", err)
	}

	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("fail to parse diff: %w", err)
	}

	return w.runFromResult(ctx, results, filediffs, w.d.Strip(), w.failOnError)
}
