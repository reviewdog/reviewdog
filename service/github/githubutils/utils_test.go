package githubutils

import (
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestLinkedMarkdownDiagnostic(t *testing.T) {
	tests := []struct {
		owner, repo, sha string
		d                *rdf.Diagnostic
		want             string
	}{
		{
			owner: "o",
			repo:  "r",
			sha:   "s",
			d: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "path/to/file.txt",
					Range: &rdf.Range{Start: &rdf.Position{
						Line:   1414,
						Column: 14,
					}},
				},
				Message: "msg",
			},
			want: "[path/to/file.txt|1414 col 14|](http://github.com/o/r/blob/s/path/to/file.txt#L1414) msg",
		},
		{
			owner: "o",
			repo:  "r",
			sha:   "s",
			d: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "path/to/file.txt",
					Range: &rdf.Range{Start: &rdf.Position{
						Line:   1414,
						Column: 0,
					}},
				},
				Message: "msg",
			},
			want: "[path/to/file.txt|1414|](http://github.com/o/r/blob/s/path/to/file.txt#L1414) msg",
		},
		{
			owner: "o",
			repo:  "r",
			sha:   "s",
			d: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: "path/to/file.txt",
				},
				Message: "msg",
			},
			want: "[path/to/file.txt||](http://github.com/o/r/blob/s/path/to/file.txt) msg",
		},
	}
	for _, tt := range tests {
		if got := LinkedMarkdownDiagnostic(tt.owner, tt.repo, tt.sha, tt.d); got != tt.want {
			t.Errorf("LinkedMarkdownDiagnostic(%q, %q, %q, %#v) = %q, want %q",
				tt.owner, tt.repo, tt.sha, tt.d, got, tt.want)
		}
	}
}
