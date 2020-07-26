package commentutil

import (
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestCommentBody(t *testing.T) {
	tests := []struct {
		in   *reviewdog.Comment
		want string
	}{
		{
			in: &reviewdog.Comment{
				ToolName: "tool-name",
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Message: "test message 1",
					},
				},
			},
			want: `
**[tool-name]** <sub>reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:</sub><br>test message 1
`,
		},
		{
			in: &reviewdog.Comment{
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Message: "test message 2 (no tool)",
					},
				},
			},
			want: `
<sub>reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:</sub><br>test message 2 (no tool)
`,
		},
		{
			in: &reviewdog.Comment{
				ToolName: "global-tool-name",
				Result: &filter.FilteredDiagnostic{
					Diagnostic: &rdf.Diagnostic{
						Message: "test message 3",
						Source:  &rdf.Source{Name: "custom-tool-name"},
					},
				},
			},
			want: `
**[custom-tool-name]** <sub>reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:</sub><br>test message 3
`,
		},
	}
	for _, tt := range tests {
		want := strings.Trim(tt.want, "\n")
		if got := MarkdownComment(tt.in); got != want {
			t.Errorf("got unexpected comment.\ngot:\n%s\nwant:\n%s", got, want)
		}
	}
}
