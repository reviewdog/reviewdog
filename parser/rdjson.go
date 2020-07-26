package parser

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/reviewdog/reviewdog/proto/rdf"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ Parser = &RDJSONParser{}

// RDJSONParser is parser for rdjsonl format.
type RDJSONParser struct{}

func NewRDJSONParser() *RDJSONParser {
	return &RDJSONParser{}
}

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
	}
	return dr.Diagnostics, nil
}
