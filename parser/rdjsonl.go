package parser

import (
	"bufio"
	"bytes"
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

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Parse parses rdjson (JSONL of Diagnostic).
func (p *RDJSONLParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	var results []*rdf.Diagnostic
	s := bufio.NewScanner(decodeBOM(r))
	for s.Scan() {
		// A UTF-8 BOM can also start a later line when files are concatenated
		// or several outputs are joined (e.g. stdout and stderr in project mode).
		line := bytes.TrimPrefix(s.Bytes(), utf8BOM)
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		d := new(rdf.Diagnostic)
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(line, d); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rdjsonl (Diagnostic): %w", err)
		}
		if d.GetOriginalOutput() == "" {
			// TODO(haya14busa): Refactor not to fill in original output.
			d.OriginalOutput = string(line)
		}
		results = append(results, d)
	}
	return results, s.Err()
}
