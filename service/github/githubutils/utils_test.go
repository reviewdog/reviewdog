package githubutils

import (
	"testing"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestLinkedMarkdownCheckResult(t *testing.T) {
	tests := []struct {
		owner, repo, sha string
		c                *reviewdog.CheckResult
		want             string
	}{
		{
			owner: "o",
			repo:  "r",
			sha:   "s",
			c: &reviewdog.CheckResult{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "path/to/file.txt",
						Range: &rdf.Range{Start: &rdf.Position{
							Line:   1414,
							Column: 14,
						}},
					},
					Message: "msg",
				},
			},
			want: "[path/to/file.txt|1414 col 14|](http://github.com/o/r/blob/s/path/to/file.txt#L1414) msg",
		},
		{
			owner: "o",
			repo:  "r",
			sha:   "s",
			c: &reviewdog.CheckResult{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "path/to/file.txt",
						Range: &rdf.Range{Start: &rdf.Position{
							Line:   1414,
							Column: 0,
						}},
					},
					Message: "msg",
				},
			},
			want: "[path/to/file.txt|1414|](http://github.com/o/r/blob/s/path/to/file.txt#L1414) msg",
		},
		{
			owner: "o",
			repo:  "r",
			sha:   "s",
			c: &reviewdog.CheckResult{
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path: "path/to/file.txt",
					},
					Message: "msg",
				},
			},
			want: "[path/to/file.txt||](http://github.com/o/r/blob/s/path/to/file.txt) msg",
		},
	}
	for _, tt := range tests {
		if got := LinkedMarkdownCheckResult(tt.owner, tt.repo, tt.sha, tt.c); got != tt.want {
			t.Errorf("LinkedMarkdownCheckResult(%q, %q, %q, %#v) = %q, want %q",
				tt.owner, tt.repo, tt.sha, tt.c, got, tt.want)
		}
	}
}
