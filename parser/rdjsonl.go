package parser

import (
	"bufio"
	"io"

	"github.com/reviewdog/reviewdog/proto/rdf"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ Parser = &RDJSONLParser{}

// RDJSONLParser is parser for rdjsonl format.
type RDJSONLParser struct{}

func NewRDJSONLParser() *RDJSONLParser {
	return &RDJSONLParser{}
}

func (p *RDJSONLParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	var results []*rdf.Diagnostic
	s := bufio.NewScanner(r)
	for s.Scan() {
		d := new(rdf.Diagnostic)
		if err := protojson.Unmarshal(s.Bytes(), d); err != nil {
			return nil, err
		}
		if d.GetOriginalOutput() == "" {
			// TODO(haya14busa): Refactor not to fill in original output.
			d.OriginalOutput = s.Text()
		}
		results = append(results, d)
	}
	return results, nil
}
