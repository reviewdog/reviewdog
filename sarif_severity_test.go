package reviewdog

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/haya14busa/go-sarif/sarif"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestSARIFCommentWriterInfoSeverity(t *testing.T) {
	var out bytes.Buffer
	w := NewSARIFCommentWriter(&out, "lint")
	if err := w.Post(context.Background(), &Comment{Result: &filter.FilteredDiagnostic{
		Diagnostic: &rdf.Diagnostic{Message: "informational finding", Severity: rdf.Severity_INFO},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got sarif.Sarif
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	level := got.Runs[0].Results[0].Level
	if level == nil || *level != sarif.Note {
		t.Fatalf("info result level = %v, want note", level)
	}
}
