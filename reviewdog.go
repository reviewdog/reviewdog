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
	"github.com/reviewdog/reviewdog/pathutil"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

// Reviewdog represents review dog application which parses result of compiler
// or linter, get diff and filter the results by diff, and report filtered
// results.
type Reviewdog struct {
	toolname   string
	p          parser.Parser
	c          CommentService
	d          DiffService
	filterMode filter.Mode
	failLevel  FailLevel
}

// NewReviewdog returns a new Reviewdog.
func NewReviewdog(toolname string, p parser.Parser, c CommentService, d DiffService, filterMode filter.Mode, failLevel FailLevel) *Reviewdog {
	return &Reviewdog{p: p, c: c, d: d, toolname: toolname, filterMode: filterMode, failLevel: failLevel}
}

// RunFromResult creates a new Reviewdog and runs it with check results.
func RunFromResult(ctx context.Context, c CommentService, results []*rdf.Diagnostic,
	filediffs []*diff.FileDiff, strip int, toolname string, filterMode filter.Mode, failLevel FailLevel) error {
	return (&Reviewdog{c: c, toolname: toolname, filterMode: filterMode, failLevel: failLevel}).runFromResult(ctx, results, filediffs, strip)
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

// FilteredCommentService is an interface which support posting filtered Comment.
type FilteredCommentService interface {
	CommentService
	PostFiltered(context.Context, *Comment) error
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
	filediffs []*diff.FileDiff, strip int) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	pathutil.NormalizePathInResults(results, wd)

	checks := filter.FilterCheck(results, filediffs, strip, wd, w.filterMode)
	shouldFail := false

	for _, check := range checks {
		comment := &Comment{
			Result:   check,
			ToolName: w.toolname,
		}
		if !check.ShouldReport {
			if fc, ok := w.c.(FilteredCommentService); ok {
				if err := fc.PostFiltered(ctx, comment); err != nil {
					return err
				}
			} else {
				continue
			}
		} else {
			if err := w.c.Post(ctx, comment); err != nil {
				return err
			}
			shouldFail = shouldFail || w.failLevel.ShouldFail(check.Diagnostic.GetSeverity())
		}
	}

	if bulk, ok := w.c.(BulkCommentService); ok {
		if err := bulk.Flush(ctx); err != nil {
			return err
		}
	}

	if shouldFail {
		return fmt.Errorf("found at least one issue with severity greater than or equal to the given level: %s", w.failLevel.String())
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

	return w.runFromResult(ctx, results, filediffs, w.d.Strip())
}
