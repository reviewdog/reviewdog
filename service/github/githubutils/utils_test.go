package githubutils

import (
	"os"
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestLinkedMarkdownDiagnostic(t *testing.T) {
	tests := []struct {
		owner, repo, sha string
		d                *rdf.Diagnostic
		serverUrl	     string
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
			serverUrl: "",
			want: "[path/to/file.txt|1414 col 14|](https://github.com/o/r/blob/s/path/to/file.txt#L1414) msg",
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
			serverUrl: "",
			want: "[path/to/file.txt|1414|](https://github.com/o/r/blob/s/path/to/file.txt#L1414) msg",
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
			serverUrl: "",
			want: "[path/to/file.txt||](https://github.com/o/r/blob/s/path/to/file.txt) msg",
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
			serverUrl: "https://xpto.com",
			want: "[path/to/file.txt||](https://xpto.com/o/r/blob/s/path/to/file.txt) msg",
		},
	}
	for _, tt := range tests {
		if errUnset := os.Unsetenv("GITHUB_SERVER_URL"); errUnset != nil {
			t.Error(errUnset)
		}
		if tt.serverUrl != "" {
			if errSet := os.Setenv("GITHUB_SERVER_URL", tt.serverUrl); errSet != nil {
				t.Error(errSet)
			}
		}
		if got := LinkedMarkdownDiagnostic(tt.owner, tt.repo, tt.sha, tt.d); got != tt.want {
			t.Errorf("LinkedMarkdownDiagnostic(%q, %q, %q, %#v) = %q, want %q",
				tt.owner, tt.repo, tt.sha, tt.d, got, tt.want)
		}
	}
}
