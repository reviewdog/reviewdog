package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestNewErrorformatParserString(t *testing.T) {
	in := []string{`%f:%l:%c:%m`, `%-G%.%#`}

	got, err := NewErrorformatParserString(in)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.efm.Efms) != len(in) {
		t.Errorf("NewErrorformatParserString: len: got %v, want %v", len(got.efm.Efms), len(in))
	}
}
func ExampleErrorformatParser() {
	const sample = `/path/to/file1.txt:1:14: [E][RULE:14] message 1
/path/to/file2.txt:2:14: [N][RULE:7] message 2`

	p, err := NewErrorformatParserString([]string{`%f:%l:%c: [%t][RULE:%n] %m`})
	if err != nil {
		panic(err)
	}
	diagnostics, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		panic(err)
	}
	for _, d := range diagnostics {
		rdjson, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(d)
		var out bytes.Buffer
		json.Indent(&out, rdjson, "", "  ")
		fmt.Println(out.String())
	}
	// Output:
	// {
	//   "message": "message 1",
	//   "location": {
	//     "path": "/path/to/file1.txt",
	//     "range": {
	//       "start": {
	//         "line": 1,
	//         "column": 14
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "code": {
	//     "value": "14"
	//   },
	//   "originalOutput": "/path/to/file1.txt:1:14: [E][RULE:14] message 1"
	// }
	// {
	//   "message": "message 2",
	//   "location": {
	//     "path": "/path/to/file2.txt",
	//     "range": {
	//       "start": {
	//         "line": 2,
	//         "column": 14
	//       }
	//     }
	//   },
	//   "severity": "INFO",
	//   "code": {
	//     "value": "7"
	//   },
	//   "originalOutput": "/path/to/file2.txt:2:14: [N][RULE:7] message 2"
	// }
}
