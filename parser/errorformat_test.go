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

func TestErrorformatParser_ParseStreams(t *testing.T) {
	efm := []string{"%E%f:%l: %m", "%C  %m", "%ZEND"}
	p, err := NewErrorformatParserString(efm)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("empty streams", func(t *testing.T) {
		ds, err := p.ParseStreams()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ds) != 0 {
			t.Errorf("got %d diagnostics, want 0", len(ds))
		}

		ds, err = p.ParseStreams(strings.NewReader(""), strings.NewReader(""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ds) != 0 {
			t.Errorf("got %d diagnostics, want 0", len(ds))
		}
	})

	t.Run("multiline diagnostics within individual streams", func(t *testing.T) {
		stream1 := strings.NewReader("a.go:1: error A\n  detail A\nEND\n")
		stream2 := strings.NewReader("b.go:2: error B\n  detail B\nEND\n")

		ds, err := p.ParseStreams(stream1, stream2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ds) != 2 {
			t.Fatalf("got %d diagnostics, want 2", len(ds))
		}
		if ds[0].Location.Path != "a.go" || ds[0].Location.Range.Start.Line != 1 || ds[0].Message != "error A\ndetail A" {
			t.Errorf("ds[0] mismatch: got %+v", ds[0])
		}
		if ds[1].Location.Path != "b.go" || ds[1].Location.Range.Start.Line != 2 || ds[1].Message != "error B\ndetail B" {
			t.Errorf("ds[1] mismatch: got %+v", ds[1])
		}
	})

	t.Run("stream affinity prevents continuation attachment across stream boundaries", func(t *testing.T) {
		stream1 := strings.NewReader("a.go:1: error A\n")
		stream2 := strings.NewReader("  detail from stream2\nEND\nb.go:2: error B\n  detail B\nEND\n")

		ds, err := p.ParseStreams(stream1, stream2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ds) != 2 {
			t.Fatalf("got %d diagnostics, want 2: %#v", len(ds), ds)
		}
		if ds[0].Location.Path != "a.go" || ds[0].Location.Range.Start.Line != 1 || ds[0].Message != "error A" {
			t.Errorf("ds[0] mismatch: got %+v", ds[0])
		}
		if ds[1].Location.Path != "b.go" || ds[1].Location.Range.Start.Line != 2 || ds[1].Message != "error B\ndetail B" {
			t.Errorf("ds[1] mismatch: got %+v", ds[1])
		}
	})

	t.Run("headers on stdout and details on stderr do not attach across streams", func(t *testing.T) {
		// Mirroring issue #2838:
		// stdout:
		// a.go:1: error A
		// b.go:2: error B
		// stderr:
		//   detail A
		// END
		//   detail B
		// END
		stdout := strings.NewReader("a.go:1: error A\nb.go:2: error B\n")
		stderr := strings.NewReader("  detail A\nEND\n  detail B\nEND\n")

		ds, err := p.ParseStreams(stdout, stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ds) != 2 {
			t.Fatalf("got %d diagnostics, want 2: %#v", len(ds), ds)
		}
		if ds[0].Location.Path != "a.go" || ds[0].Location.Range.Start.Line != 1 || ds[0].Message != "error A" {
			t.Errorf("ds[0] mismatch: got %+v", ds[0])
		}
		if ds[1].Location.Path != "b.go" || ds[1].Location.Range.Start.Line != 2 || ds[1].Message != "error B" {
			t.Errorf("ds[1] mismatch: got %+v", ds[1])
		}
	})
}
