package parser

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &CheckStyleParser{}

// CheckStyleParser is checkstyle parser.
type CheckStyleParser struct{}

// NewCheckStyleParser returns a new CheckStyleParser.
func NewCheckStyleParser() Parser {
	return &CheckStyleParser{}
}

func (p *CheckStyleParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	var cs = new(CheckStyleResult)
	if err := xml.NewDecoder(r).Decode(cs); err != nil {
		return nil, err
	}
	var ds []*rdf.Diagnostic
	for _, file := range cs.Files {
		for _, cerr := range file.Errors {
			d := &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: file.Name,
					Range: &rdf.Range{
						Start: &rdf.Position{
							Line:   int32(cerr.Line),
							Column: int32(cerr.Column),
						},
					},
				},
				Message:  cerr.Message,
				Severity: severity(cerr.Severity),
				OriginalOutput: fmt.Sprintf("%v:%d:%d: %v: %v (%v)",
					file.Name, cerr.Line, cerr.Column, cerr.Severity, cerr.Message, cerr.Source),
			}
			if s := cerr.Source; s != "" {
				d.Code = &rdf.Code{Value: s}
			}
			ds = append(ds, d)
		}
	}
	return ds, nil
}

// CheckStyleResult represents checkstyle XML result.
// <?xml version="1.0" encoding="utf-8"?><checkstyle version="4.3"><file ...></file>...</checkstyle>
//
// References:
//   - http://checkstyle.sourceforge.net/
//   - http://eslint.org/docs/user-guide/formatters/#checkstyle
type CheckStyleResult struct {
	XMLName xml.Name          `xml:"checkstyle"`
	Version string            `xml:"version,attr"`
	Files   []*CheckStyleFile `xml:"file,omitempty"`
}

// CheckStyleFile represents <file name="fname"><error ... />...</file>
type CheckStyleFile struct {
	Name   string             `xml:"name,attr"`
	Errors []*CheckStyleError `xml:"error"`
}

// CheckStyleError represents <error line="1" column="10" severity="error" message="msg" source="src" />
type CheckStyleError struct {
	Column   int    `xml:"column,attr,omitempty"`
	Line     int    `xml:"line,attr"`
	Message  string `xml:"message,attr"`
	Severity string `xml:"severity,attr,omitempty"`
	Source   string `xml:"source,attr,omitempty"`
}
