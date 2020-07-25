package parser

import (
	"io"
	"strings"

	"github.com/reviewdog/errorformat"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &ErrorformatParser{}

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
	var rs []*rdf.Diagnostic
	for s.Scan() {
		e := s.Entry()
		if e.Valid {
			rs = append(rs, &rdf.Diagnostic{
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
				OriginalOutput: strings.Join(e.Lines, "\n"),
			})
		}
	}
	return rs, nil
}
