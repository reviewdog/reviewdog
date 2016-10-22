package reviewdog

import (
	"io"

	"github.com/haya14busa/errorformat"
)

var _ Parser = &ErrorformatParser{}

// ErrorformatParser is errorformat parser.
type ErrorformatParser struct {
	efm *errorformat.Errorformat
}

// NewErrorformatParser returns a new ErrorformatParser.
func NewErrorformatParser(efm *errorformat.Errorformat) Parser {
	return &ErrorformatParser{efm: efm}
}

func (p *ErrorformatParser) Parse(r io.Reader) ([]*CheckResult, error) {
	s := p.efm.NewScanner(r)
	var rs []*CheckResult
	for s.Scan() {
		e := s.Entry()
		if e.Valid {
			rs = append(rs, &CheckResult{
				Path:    e.Filename,
				Lnum:    e.Lnum,
				Col:     e.Col,
				Message: e.Text,
				Lines:   e.Lines,
			})
		}
	}
	return rs, nil
}
