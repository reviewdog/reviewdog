package parser

import (
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestNewErrorformatParserString(t *testing.T) {
	in := []string{`%f:%l:%c:%m`, `%-G%.%#`}

	got, err := NewErrorformatParserString(in)
	assert.NoError(t, err)

	if len(got.efm.Efms) != len(in) {
		t.Errorf("NewErrorformatParserString: len: got %v, want %v", len(got.efm.Efms), len(in))
	}
}

func TestErrorformatParser(t *testing.T) {
	const sample = `/path/to/file1.txt:1:14: [E][RULE:14] message 1
/path/to/file2.txt:2:14: [N][RULE:7] message 2`

	p, err := NewErrorformatParserString([]string{`%f:%l:%c: [%t][RULE:%n] %m`})
	assert.NoError(t, err)
	gotDiagnostics, err := p.Parse(strings.NewReader(sample))
	assert.NoError(t, err)

	wantDiagnostics := []*rdf.Diagnostic{
		&rdf.Diagnostic{
			Message: "message 1",
			Location: &rdf.Location{
				Path: "/path/to/file1.txt",
				Range: &rdf.Range{
					Start: &rdf.Position{
						Line:   1,
						Column: 14,
					},
				},
			},
			Severity: rdf.Severity_ERROR,
			Code: &rdf.Code{
				Value: "14",
			},
			OriginalOutput: "/path/to/file1.txt:1:14: [E][RULE:14] message 1",
		},
		&rdf.Diagnostic{
			Message: "message 2",
			Location: &rdf.Location{
				Path: "/path/to/file2.txt",
				Range: &rdf.Range{
					Start: &rdf.Position{
						Line:   2,
						Column: 14,
					},
				},
			},
			Severity: rdf.Severity_INFO,
			Code:     &rdf.Code{
				Value: "7",
			},
			OriginalOutput: "/path/to/file2.txt:2:14: [N][RULE:7] message 2",
		},
	}
	if diff := cmp.Diff(wantDiagnostics, gotDiagnostics, protocmp.Transform()); diff != "" {
		t.Errorf("Error format parsing returned diff (-want +got):\n%s", diff)
	}
}

func TestErrorformatParserMultiline(t *testing.T) {
	const sample = `path/to/file1.txt:1: category-a: First line of message for category-a.
Second line of message for category-a.
path/to/file1.txt:11: category-b: Message 11 for category-b.`

	p, err := NewErrorformatParserString([]string{`%A%f:%l: %m`, `%+C%m`})
	assert.NoError(t, err)
	gotDiagnostics, err := p.Parse(strings.NewReader(sample))
	assert.NoError(t, err)

	wantDiagnostics := []*rdf.Diagnostic{
		&rdf.Diagnostic{
			Message: "category-a: First line of message for category-a.\nSecond line of message for category-a.",
			Location: &rdf.Location{
				Path: "path/to/file1.txt",
				Range: &rdf.Range{
					Start: &rdf.Position{
						Line:   1,
					},
				},
			},
			OriginalOutput: "path/to/file1.txt:1: category-a: First line of message for category-a.\nSecond line of message for category-a.",
		},
		&rdf.Diagnostic{
			Message: "category-b: Message 11 for category-b.",
			Location: &rdf.Location{
				Path: "path/to/file1.txt",
				Range: &rdf.Range{
					Start: &rdf.Position{
						Line:   11,
					},
				},
			},
			OriginalOutput: "path/to/file1.txt:11: category-b: Message 11 for category-b.",
		},
	}
	if diff := cmp.Diff(wantDiagnostics, gotDiagnostics, protocmp.Transform()); diff != "" {
		t.Errorf("Error format parsing returned diff (-want +got):\n%s", diff)
	}
}
