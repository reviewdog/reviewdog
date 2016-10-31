package reviewdog

import (
	"encoding/xml"
	"fmt"
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

var _ Parser = &CheckStyleParser{}

// CheckStyleParser is checkstyle parser.
type CheckStyleParser struct{}

// NewCheckStyleParser returns a new CheckStyleParser.
func NewCheckStyleParser() Parser {
	return &CheckStyleParser{}
}

func (p *CheckStyleParser) Parse(r io.Reader) ([]*CheckResult, error) {
	var cs = new(CheckStyleResult)
	if err := xml.NewDecoder(r).Decode(cs); err != nil {
		return nil, err
	}
	var rs []*CheckResult
	for _, file := range cs.Files {
		for _, cerr := range file.Errors {
			rs = append(rs, &CheckResult{
				Path:    file.Name,
				Lnum:    cerr.Line,
				Col:     cerr.Column,
				Message: cerr.Message,
				Lines: []string{
					fmt.Sprintf("%v:%d:%d: %v: %v (%v)",
						file.Name, cerr.Line, cerr.Column, cerr.Severity, cerr.Message, cerr.Source),
				},
			})
		}
	}
	return rs, nil
}

type CheckStyleResult struct {
	XMLName xml.Name          `xml:"checkstyle"`
	Version string            `xml:"version,attr"`
	Files   []*CheckStyleFile `xml:"file,omitempty"`
}

type CheckStyleFile struct {
	Name   string             `xml:"name,attr"`
	Errors []*CheckStyleError `xml:"error"`
}

type CheckStyleError struct {
	Column   int    `xml:"column,attr,omitempty"`
	Line     int    `xml:"line,attr"`
	Message  string `xml:"message,attr"`
	Severity string `xml:"severity,attr,omitempty"`
	Source   string `xml:"source,attr,omitempty"`
}
