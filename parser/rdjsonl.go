package parser

import (
	"bufio"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &RDJSONLParser{}

// RDJSONLParser is parser for rdjsonl format.
type RDJSONLParser struct{}

// NewRDJSONLParser returns a new RDJSONParser.
func NewRDJSONLParser() *RDJSONLParser {
	return &RDJSONLParser{}
}

// Parse parses rdjson (JSONL of Diagnostic).
func (p *RDJSONLParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	var results []*rdf.Diagnostic
	s := bufio.NewScanner(r)
	for s.Scan() {
		d := new(rdf.Diagnostic)
		if err := protojson.Unmarshal(s.Bytes(), d); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rdjsonl (Diagnostic): %w", err)
		}
		if d.GetOriginalOutput() == "" {
			// TODO(haya14busa): Refactor not to fill in original output.
			d.OriginalOutput = s.Text()
		}
		results = append(results, d)
	}
	return results, nil
}
