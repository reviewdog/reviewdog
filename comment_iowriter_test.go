package reviewdog

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestUnifiedCommentWriter_Post(t *testing.T) {
	tests := []struct {
		in   *Comment
		want string
	}{
		{
			in: &Comment{
				Result: &FilteredCheck{
					Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{Path: "/path/to/file"}},
				},
				ToolName: "tool name",
				Body:     "message",
			},
			want: `/path/to/file: [tool name] message`,
		},
		{
			in: &Comment{
				Result: &FilteredCheck{
					Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{
						Path: "/path/to/file",
						Range: &rdf.Range{Start: &rdf.Position{
							Column: 14,
						}},
					}},
				},
				ToolName: "tool name",
				Body:     "message",
			},
			want: `/path/to/file: [tool name] message`,
		},
		{
			in: &Comment{
				Result: &FilteredCheck{
					Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{
						Path: "/path/to/file",
						Range: &rdf.Range{Start: &rdf.Position{
							Line: 14,
						}},
					}},
				},
				ToolName: "tool name",
				Body:     "message",
			},
			want: `/path/to/file:14: [tool name] message`,
		},
		{
			in: &Comment{
				Result: &FilteredCheck{
					Diagnostic: &rdf.Diagnostic{Location: &rdf.Location{
						Path: "/path/to/file",
						Range: &rdf.Range{Start: &rdf.Position{
							Line:   14,
							Column: 7,
						}},
					}},
				},
				ToolName: "tool name",
				Body:     "line1\nline2",
			},
			want: `/path/to/file:14:7: [tool name] line1
line2`,
		},
	}
	for _, tt := range tests {
		buf := new(bytes.Buffer)
		mc := NewUnifiedCommentWriter(buf)
		if err := mc.Post(context.Background(), tt.in); err != nil {
			t.Error(err)
			continue
		}
		if got := strings.Trim(buf.String(), "\n"); got != tt.want {
			t.Errorf("UnifiedCommentWriter_Post(%v) = \n%v\nwant:\n%v", tt.in, got, tt.want)
		}
	}
}
