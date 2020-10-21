package parser

import (
	"fmt"
	"io"
	"io/ioutil"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &RDJSONParser{}

// RDJSONParser is parser for rdjsonl format.
type RDJSONParser struct{}

// NewRDJSONParser returns a new RDJSONParser.
func NewRDJSONParser() *RDJSONParser {
	return &RDJSONParser{}
}

// Parse parses rdjson (JSON of DiagnosticResult).
func (p *RDJSONParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var dr rdf.DiagnosticResult
	if err := protojson.Unmarshal(b, &dr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rdjson (DiagnosticResult): %w", err)
	}
	for _, d := range dr.Diagnostics {
		// Fill in default severity and source for each diagnostic.
		if d.Severity == rdf.Severity_UNKNOWN_SEVERITY {
			d.Severity = dr.GetSeverity()
		}
		if d.Source == nil {
			d.Source = dr.Source
		}
		if d.GetOriginalOutput() == "" {
			// TODO(haya14busa): Refactor not to fill in original output.
			d.OriginalOutput = d.String()
		}
	}
	return dr.Diagnostics, nil
}
