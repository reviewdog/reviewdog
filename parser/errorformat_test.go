package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

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

// A line the scanner cannot hold must not end the parse: every diagnostic
// around it is still reported, and the long line itself is cut.
func TestErrorformatParser_Parse_longLine(t *testing.T) {
	p, err := NewErrorformatParserString([]string{`%f:%l:%c: %m`})
	if err != nil {
		t.Fatal(err)
	}
	atLimit := "c.go:3:3: " + strings.Repeat("x", maxLineLen-len("c.go:3:3: "))
	in := "a.go:1:1: first\n" +
		"b.go:2:2: " + strings.Repeat("y", 3*bufio.MaxScanTokenSize) + "\n" +
		atLimit + "\n" +
		atLimit + "\r\n" +
		"d.go:4:4: last" // no trailing newline

	got, err := p.Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{"a.go", "b.go", "c.go", "c.go", "d.go"}
	if len(got) != len(wantPaths) {
		t.Fatalf("got %d diagnostics, want %d", len(got), len(wantPaths))
	}
	for i, d := range got {
		if path := d.GetLocation().GetPath(); path != wantPaths[i] {
			t.Errorf("diagnostic %d: path = %q, want %q", i, path, wantPaths[i])
		}
	}
	if got, want := len(got[1].GetMessage()), maxLineLen-len("b.go:2:2: "); got != want {
		t.Errorf("long line message length = %d, want it cut to %d", got, want)
	}
	if got[2].GetMessage() != strings.Repeat("x", maxLineLen-len("c.go:3:3: ")) {
		t.Errorf("a line of exactly maxLineLen bytes must not be changed")
	}
}

func TestLineLimitReader(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{name: "empty", in: "", max: 3, want: ""},
		{name: "short lines are unchanged", in: "ab\ncd\n", max: 3, want: "ab\ncd\n"},
		{name: "line at the limit", in: "abc\ndef", max: 3, want: "abc\ndef"},
		{name: "long line is cut", in: "abcdef\ngh\n", max: 3, want: "abc\ngh\n"},
		{name: "cut applies to every line", in: "abcd\nefgh\nij", max: 2, want: "ab\nef\nij"},
		{name: "empty lines are kept", in: "abcd\n\n\nef", max: 2, want: "ab\n\n\nef"},
		{name: "long last line without newline", in: "ab\ncdefgh", max: 3, want: "ab\ncde"},
		{name: "crlf counts toward the limit", in: "abcd\r\nef\r\n", max: 3, want: "abc\nef\r\n"},
	}
	for _, tt := range tests {
		// Read in one go and one byte at a time: the cut must not depend on how
		// the underlying reader splits the input.
		for name, wrap := range map[string]func(io.Reader) io.Reader{
			"whole":    func(r io.Reader) io.Reader { return r },
			"one byte": iotest.OneByteReader,
		} {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				lr := &lineLimitReader{r: wrap(strings.NewReader(tt.in)), max: tt.max}
				got, err := io.ReadAll(lr)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
				if wantCut := tt.want != tt.in; lr.truncated != wantCut {
					t.Errorf("truncated = %v, want %v", lr.truncated, wantCut)
				}
			})
		}
	}
}
