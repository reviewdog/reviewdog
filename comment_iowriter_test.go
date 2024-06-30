package reviewdog

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestUnifiedCommentWriter_Post(t *testing.T) {
	tests := []struct {
		in   *Comment
		want string
	}{
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{Path: "/path/to/file"},
						Message:  "message",
					},
				},
				ToolName: "tool name",
			},
			want: `/path/to/file: [tool name] message`,
		},
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path: "/path/to/file",
							Range: &rdf.Range{Start: &rdf.Position{
								Column: 14,
							}},
						},
						Message: "message",
					},
				},
				ToolName: "tool name",
			},
			want: `/path/to/file: [tool name] message`,
		},
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path: "/path/to/file",
							Range: &rdf.Range{Start: &rdf.Position{
								Line: 14,
							}},
						},
						Message: "message",
					},
				},
				ToolName: "tool name",
			},
			want: `/path/to/file:14: [tool name] message`,
		},
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path: "/path/to/file",
							Range: &rdf.Range{Start: &rdf.Position{
								Line:   14,
								Column: 7,
							}},
						},
						Message: "line1\nline2",
					},
				},
				ToolName: "tool name",
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

func TestRDJSONLCommentWriter_Post(t *testing.T) {
	tests := []struct {
		in   *Comment
		want string
	}{
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{Path: "/path/to/file"},
						Message:  "message",
					},
				},
				ToolName: "tool name",
			},
			want: `{"message":"message","location":{"path":"/path/to/file"},"source":{"name":"tool name"}}`,
		},
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path: "/path/to/file",
							Range: &rdf.Range{Start: &rdf.Position{
								Column: 14,
							}},
						},
						Message: "message",
					},
				},
			},
			want: `{"message":"message","location":{"path":"/path/to/file","range":{"start":{"column":14}}}}`,
		},
		{
			in: &Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Location: &rdf.Location{
							Path: "/path/to/file",
							Range: &rdf.Range{Start: &rdf.Position{
								Column: 14,
							}},
						},
						Message: "message",
						Source: &rdf.Source{
							Name: "tool name in Diagnostic",
							Url:  "tool url",
						},
					},
				},
			},
			want: `{"message":"message","location":{"path":"/path/to/file","range":{"start":{"column":14}}},"source":{"name":"tool name in Diagnostic","url":"tool url"}}`,
		},
	}
	for _, tt := range tests {
		buf := new(bytes.Buffer)
		cw := NewRDJSONLCommentWriter(buf)
		if err := cw.Post(context.Background(), tt.in); err != nil {
			t.Error(err)
			continue
		}
		if got := strings.ReplaceAll(strings.Trim(buf.String(), "\n"), `, "`, `,"`); got != tt.want {
			t.Errorf("RDJSONLCommentWriter.Post(%v) = \n%v\nwant:\n%v", tt.in, got, tt.want)
		}
	}
}

func TestRDJSONCommentWriter_Post(t *testing.T) {
	comments := []*Comment{
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{Path: "/path/to/file"},
					Message:  "message",
				},
			},
			ToolName: "tool name",
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "/path/to/file",
						Range: &rdf.Range{Start: &rdf.Position{
							Column: 14,
						}},
					},
					Message: "message",
				},
			},
		},
		{
			Result: &filter.FilteredDiagnostic{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "/path/to/file",
						Range: &rdf.Range{Start: &rdf.Position{
							Column: 14,
						}},
					},
					Message: "message",
					Source: &rdf.Source{
						Name: "tool name in Diagnostic",
						Url:  "tool url",
					},
				},
			},
		},
	}
	buf := new(bytes.Buffer)
	cw := NewRDJSONCommentWriter(buf, "tool name [constructor]")
	for _, c := range comments {
		if err := cw.Post(context.Background(), c); err != nil {
			t.Error(err)
		}
	}
	if err := cw.Flush(context.Background()); err != nil {
		t.Error(err)
	}
	want := `{
  "diagnostics": [
    {
      "message": "message",
      "location": {
        "path": "/path/to/file"
      }
    },
    {
      "message": "message",
      "location": {
        "path": "/path/to/file",
        "range": {
          "start": {
            "column": 14
          }
        }
      }
    },
    {
      "message": "message",
      "location": {
        "path": "/path/to/file",
        "range": {
          "start": {
            "column": 14
          }
        }
      },
      "source": {
        "name": "tool name in Diagnostic",
        "url": "tool url"
      }
    }
  ],
  "source": {
    "name": "tool name [constructor]"
  }
}`
	re := regexp.MustCompile(`:\s+`)
	got := re.ReplaceAllString(strings.TrimSpace(buf.String()), ":")
	if got != re.ReplaceAllString(strings.TrimSpace(want), ":") {
		t.Errorf("got\n%v\nwant:\n%v", got, want)
	}
}
