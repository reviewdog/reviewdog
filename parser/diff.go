package parser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &DiffParser{}

// DiffParser is a unified diff parser.
type DiffParser struct {
	strip int
	wd    string
}

// NewDiffParser creates a new DiffParser.
func NewDiffParser(strip int) *DiffParser {
	p := &DiffParser{strip: strip}
	p.wd, _ = os.Getwd()
	return p
}

// Parse parses input as unified diff format and return it as diagnostics.
func (p *DiffParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	filediffs, err := diff.ParseMultiFile(r)
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %w", err)
	}
	diagnostics := []*rdf.Diagnostic{}
	for _, fdiff := range filediffs {
		path := filter.NormalizeDiffPath(fdiff.PathNew, p.strip)
		for _, hunk := range fdiff.Hunks {
			lnum := hunk.StartLineOld - 1
			prevState := diff.LineUnchanged
			var (
				startLine     int
				isInsert      bool
				newLines      []string
				originalLines []string // For Diagnostic.original_output
			)
			reset := func() {
				startLine = 0
				isInsert = false
				newLines = []string{}
				originalLines = []string{}
			}
			emit := func() {
				drange := &rdf.Range{ // Diagnostic Range
					Start: &rdf.Position{Line: int32(startLine)},
					End:   &rdf.Position{Line: int32(lnum)},
				}
				text := strings.Join(newLines, "\n")
				if isInsert {
					text += "\n" // Need line-break at the end if it's insertion,
					drange.GetEnd().Line = int32(startLine)
					drange.GetEnd().Column = 1
					drange.GetStart().Column = 1
				}
				d := &rdf.Diagnostic{
					Location:       &rdf.Location{Path: path, Range: drange},
					Suggestions:    []*rdf.Suggestion{{Range: drange, Text: text}},
					OriginalOutput: strings.Join(originalLines, "\n"),
				}
				diagnostics = append(diagnostics, d)
				reset()
			}
			for i, diffLine := range hunk.Lines {
				switch diffLine.Type {
				case diff.LineAdded:
					if i == 0 {
						lnum++ // Increment line number only when it's at head.
					}
					newLines = append(newLines, diffLine.Content)
					originalLines = append(originalLines, buildOriginalLine(path, diffLine))
					switch prevState {
					case diff.LineUnchanged:
						// Insert.
						startLine = lnum + 1
						isInsert = true
					case diff.LineDeleted, diff.LineAdded:
						// Do nothing in particular.
					}
				case diff.LineDeleted:
					lnum++
					originalLines = append(originalLines, buildOriginalLine(path, diffLine))
					switch prevState {
					case diff.LineUnchanged:
						startLine = lnum
					case diff.LineAdded:
						isInsert = false
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
			if startLine > 0 {
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
