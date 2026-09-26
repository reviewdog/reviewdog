package commentutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestBuildSuggestionText(t *testing.T) {
	tests := []struct {
		name        string
		sourceLines map[int]string
		suggestion  *rdf.Suggestion
		want        string
		wantErr     string
	}{
		{
			name:        "same line",
			sourceLines: map[int]string{1: "hello world"},
			suggestion: &rdf.Suggestion{
				Range: &rdf.Range{
					Start: &rdf.Position{Line: 1, Column: 7},
					End:   &rdf.Position{Line: 1, Column: 12},
				},
				Text: "there",
			},
			want: "hello there",
		},
		{
			name:        "multiple lines",
			sourceLines: map[int]string{1: "old prefix", 2: "tail text"},
			suggestion: &rdf.Suggestion{
				Range: &rdf.Range{
					Start: &rdf.Position{Line: 1, Column: 5},
					End:   &rdf.Position{Line: 2, Column: 5},
				},
				Text: "NEW",
			},
			want: "old NEW text",
		},
		{
			name: "no source lines",
			suggestion: &rdf.Suggestion{
				Range: &rdf.Range{
					Start: &rdf.Position{Line: 1},
					End:   &rdf.Position{Line: 1},
				},
			},
			wantErr: "source lines are not available",
		},
		{
			name:        "missing source line",
			sourceLines: map[int]string{1: "hello"},
			suggestion: &rdf.Suggestion{
				Range: &rdf.Range{
					Start: &rdf.Position{Line: 1},
					End:   &rdf.Position{Line: 2},
				},
			},
			wantErr: "source line (L=2) is not available for this suggestion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSuggestionText(tt.sourceLines, tt.suggestion)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("BuildSuggestionText() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildSuggestionText() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildSuggestionText() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
