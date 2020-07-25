package reviewdog

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestMultiCommentService_Post(t *testing.T) {
	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	w := MultiCommentService(NewRawCommentWriter(buf1), NewRawCommentWriter(buf2))

	const want = "line1\nline2"

	c := &Comment{Result: &filter.FilteredDiagnostic{Diagnostic: &rdf.Diagnostic{OriginalOutput: want}}}
	if err := w.Post(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	if got := strings.Trim(buf1.String(), "\n"); got != want {
		t.Errorf("writer 1: got %v, want %v", got, want)
	}

	if got := strings.Trim(buf2.String(), "\n"); got != want {
		t.Errorf("writer 2: got %v, want %v", got, want)
	}

	if err := w.(BulkCommentService).Flush(context.Background()); err != nil {
		t.Errorf("MultiCommentService implements BulkCommentService and should not return error when any services implements it: %v", err)
	}
}

type fakeBulkCommentService struct {
	BulkCommentService
	calledFlush bool
}

func (f *fakeBulkCommentService) Flush(_ context.Context) error {
	f.calledFlush = true
	return nil
}

func TestMultiCommentService_Flush(t *testing.T) {
	f1 := &fakeBulkCommentService{}
	f2 := &fakeBulkCommentService{}
	w := MultiCommentService(f1, f2)
	if err := w.(BulkCommentService).Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !f1.calledFlush || !f2.calledFlush {
		t.Error("MultiCommentService_Flush should run Flush() for every services")
	}
}
