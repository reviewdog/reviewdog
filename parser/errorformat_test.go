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
	"unicode/utf8"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/reviewdog/reviewdog/service/serviceutil"
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

// A character that straddles the limit must be dropped whole: half a character
// is invalid UTF-8, which proto marshaling, and so Fingerprint, rejects.
func TestErrorformatParser_Parse_multibyteAtLimit(t *testing.T) {
	p, err := NewErrorformatParserString([]string{`%f:%l:%c: %m`})
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "a.go:1:1: "
	// Pad the message by 0 to 3 bytes so that every character width straddles
	// the limit at every possible offset.
	for _, char := range []string{"é", "あ", "😀"} {
		for pad := 0; pad < utf8.UTFMax; pad++ {
			t.Run(fmt.Sprintf("%s/pad %d", char, pad), func(t *testing.T) {
				msg := strings.Repeat("x", pad) + strings.Repeat(char, bufio.MaxScanTokenSize)
				in := prefix + msg + "\nb.go:2:2: after\n"

				got, err := p.Parse(strings.NewReader(in))
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != 2 || got[1].GetLocation().GetPath() != "b.go" {
					t.Fatalf("got %d diagnostics, want the long line and the one after it", len(got))
				}
				for _, d := range got {
					if !utf8.ValidString(d.GetMessage()) || !utf8.ValidString(d.GetOriginalOutput()) {
						t.Errorf("%q: invalid UTF-8 in Message or OriginalOutput", d.GetLocation().GetPath())
					}
					if _, err := serviceutil.Fingerprint(d); err != nil {
						t.Errorf("%q: Fingerprint: %v", d.GetLocation().GetPath(), err)
					}
				}
				// Only the character that did not fit is lost.
				if n := len(prefix) + len(got[0].GetMessage()); n > maxLineLen || n <= maxLineLen-len(char) {
					t.Errorf("long line is %d bytes, want %d to %d", n, maxLineLen-len(char)+1, maxLineLen)
				}
			})
		}
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
		// A multi-byte character is kept whole or dropped whole.
		{name: "character straddling the limit", in: "aあ\n", max: 2, want: "a\n"},
		{name: "character straddling the limit by one byte", in: "aあ\n", max: 3, want: "a\n"},
		{name: "character ending at the limit", in: "aあ\n", max: 4, want: "aあ\n"},
		{name: "four-byte character", in: "ab😀\n", max: 5, want: "ab\n"},
		{name: "cut after a whole character", in: "あい\n", max: 3, want: "あ\n"},
		{name: "character wider than the limit", in: "あ\nb\n", max: 2, want: "\nb\n"},
		{name: "nothing is kept after a dropped character", in: "aあb\nc\n", max: 2, want: "a\nc\n"},
		{name: "invalid bytes are passed through", in: "\x80\x80\x80\x80\n", max: 3, want: "\x80\x80\x80\n"},
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
