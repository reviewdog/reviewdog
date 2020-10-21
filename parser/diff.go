package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &DiffParser{}

// DiffParser is a unified diff parser.
type DiffParser struct {
	strip int
}

// NewDiffParser creates a new DiffParser.
func NewDiffParser(strip int) *DiffParser {
	return &DiffParser{strip: strip}
}

// state data for a diagnostic.
type dstate struct {
	startLine     int
	isInsert      bool
	newLines      []string
	originalLines []string // For Diagnostic.original_output
}

func (d dstate) build(path string, currentLine int) *rdf.Diagnostic {
	drange := &rdf.Range{ // Diagnostic Range
		Start: &rdf.Position{Line: int32(d.startLine)},
		End:   &rdf.Position{Line: int32(currentLine)},
	}
	text := strings.Join(d.newLines, "\n")
	if d.isInsert {
		text += "\n" // Need line-break at the end if it's insertion,
		drange.GetEnd().Line = int32(d.startLine)
		drange.GetEnd().Column = 1
		drange.GetStart().Column = 1
	}
	return &rdf.Diagnostic{
		Location:       &rdf.Location{Path: path, Range: drange},
		Suggestions:    []*rdf.Suggestion{{Range: drange, Text: text}},
		OriginalOutput: strings.Join(d.originalLines, "\n"),
	}
}

// Parse parses input as unified diff format and return it as diagnostics.
func (p *DiffParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	filediffs, err := diff.ParseMultiFile(r)
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %w", err)
	}
	var diagnostics []*rdf.Diagnostic
	for _, fdiff := range filediffs {
		path := filter.NormalizeDiffPath(fdiff.PathNew, p.strip)
		for _, hunk := range fdiff.Hunks {
			lnum := hunk.StartLineOld - 1
			prevState := diff.LineUnchanged
			state := dstate{}
			emit := func() {
				diagnostics = append(diagnostics, state.build(path, lnum))
				state = dstate{}
			}
			for i, diffLine := range hunk.Lines {
				switch diffLine.Type {
				case diff.LineAdded:
					if i == 0 {
						lnum++ // Increment line number only when it's at head.
					}
					state.newLines = append(state.newLines, diffLine.Content)
					state.originalLines = append(state.originalLines, buildOriginalLine(path, diffLine))
					switch prevState {
					case diff.LineUnchanged:
						// Insert.
						state.startLine = lnum + 1
						state.isInsert = true
					case diff.LineDeleted, diff.LineAdded:
						// Do nothing in particular.
					}
				case diff.LineDeleted:
					lnum++
					state.originalLines = append(state.originalLines, buildOriginalLine(path, diffLine))
					switch prevState {
					case diff.LineUnchanged:
						state.startLine = lnum
					case diff.LineAdded:
						state.isInsert = false
					case diff.LineDeleted:
						// Do nothing in particular.
					}
				case diff.LineUnchanged:
					switch prevState {
					case diff.LineUnchanged:
						// Do nothing in particular.
					case diff.LineAdded, diff.LineDeleted:
						emit() // Output a diagnostic.
					}
					lnum++
				}
				prevState = diffLine.Type
			}
			if state.startLine > 0 {
				emit() // Output a diagnostic at the end of hunk.
			}
		}
	}
	return diagnostics, nil
}

func buildOriginalLine(path string, line *diff.Line) string {
	var (
		lnum int
		mark rune
	)
	switch line.Type {
	case diff.LineAdded:
		mark = '+'
		lnum = line.LnumNew
	case diff.LineDeleted:
		mark = '-'
		lnum = line.LnumOld
	case diff.LineUnchanged:
		mark = ' '
		lnum = line.LnumOld
	}
	return fmt.Sprintf("%s:%d:%s%s", path, lnum, string(mark), line.Content)
}
