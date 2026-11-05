package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/reviewdog/errorformat"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &ErrorformatParser{}
var _ StreamParser = &ErrorformatParser{}

// ErrorformatParser is errorformat parser.
type ErrorformatParser struct {
	efm *errorformat.Errorformat
}

// NewErrorformatParser returns a new ErrorformatParser.
func NewErrorformatParser(efm *errorformat.Errorformat) *ErrorformatParser {
	return &ErrorformatParser{efm: efm}
}

// NewErrorformatParserString returns a new ErrorformatParser from errorformat
// in string representation.
func NewErrorformatParserString(efms []string) (*ErrorformatParser, error) {
	efm, err := errorformat.NewErrorformat(efms)
	if err != nil {
		return nil, err
	}
	return NewErrorformatParser(efm), nil
}

func (p *ErrorformatParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	s := p.efm.NewScanner(r)
	var ds []*rdf.Diagnostic
	for s.Scan() {
		e := s.Entry()
		if e.Valid {
			d := &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: e.Filename,
					Range: &rdf.Range{
						Start: &rdf.Position{
							Line:   int32(e.Lnum),
							Column: int32(e.Col),
						},
					},
				},
				Message:        e.Text,
				Severity:       severity(string(e.Type)),
				OriginalOutput: strings.Join(e.Lines, "\n"),
			}
			if e.Nr != 0 {
				d.Code = &rdf.Code{Value: fmt.Sprintf("%d", e.Nr)}
			}
			ds = append(ds, d)
		}
	}
	return ds, nil
}

// ParseStreams parses multiple streams sequentially, maintaining stream affinity
// so that multiline diagnostics from one stream do not absorb lines from another stream.
// It returns on the first error encountered and discards diagnostics from earlier streams.
func (p *ErrorformatParser) ParseStreams(readers ...io.Reader) ([]*rdf.Diagnostic, error) {
	var allDiags []*rdf.Diagnostic
	for _, r := range readers {
		diags, err := p.Parse(r)
		if err != nil {
			return nil, err
		}
		allDiags = append(allDiags, diags...)
	}
	return allDiags, nil
}
