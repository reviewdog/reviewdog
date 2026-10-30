package parser

import (
	"bufio"
	"strings"
	"testing"
)

func TestRDJSONLParser(t *testing.T) {
	const sample = `{"source":{"name":"deadcode"},"message":"'unused' is unused","location":{"path":"testdata/main.go","range":{"start":{"line":18,"column":6}}}}
{"source":{"name":"deadcode"},"message":"'unused2' is unused","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}}
{"source":{"name":"errcheck"},"message":"Error return value of 'os.Open' is not checked","location":{"path":"testdata/main.go","range":{"start":{"line":15,"column":9}}}}
{"source":{"name":"ineffassign"},"message":"ineffectual assignment to 'x'","location":{"path":"testdata/main.go","range":{"start":{"line":12,"column":2}}}}
{"source":{"name":"govet"},"message":"printf: Sprintf format %d reads arg #1, but call has 0 args","location":{"path":"testdata/main.go","range":{"start":{"line":13,"column":2}}}}
{"source":{"name":"severity-test"},"message":"severity test (string)","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}, "severity": "WARNING"}
{"source":{"name":"severity-test"},"message":"severity test (number)","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}, "severity": "WARNING", "unknown_field": "this field will be ignored"}`
	sampleLines := strings.Split(sample, "\n")
	p := NewRDJSONLParser()
	diagnostics, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		t.Error(err)
	}
	for i, d := range diagnostics {
		if got, want := d.GetOriginalOutput(), sampleLines[i]; got != want {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestRDJSONLParserLineTooLong(t *testing.T) {
	longLine := strings.Repeat("a", bufio.MaxScanTokenSize)
	p := NewRDJSONLParser()
	_, err := p.Parse(strings.NewReader(longLine))
	if err != bufio.ErrTooLong {
		t.Errorf("expected bufio.ErrTooLong, got %s", err)
	}
}

func TestRDJSONLParserBlankLines(t *testing.T) {
	const line = `{"message":"a","location":{"path":"main.go"}}`
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty input", "", 0},
		{"only newline", "\n", 0},
		{"whitespace only line", "  \t\n", 0},
		{"trailing blank line", line + "\n\n", 1},
		{"blank line between", line + "\n\n" + line + "\n", 2},
		{"CRLF with blank line", line + "\r\n\r\n" + line + "\r\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics, err := NewRDJSONLParser().Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if got := len(diagnostics); got != tt.want {
				t.Errorf("got %d diagnostics, want %d", got, tt.want)
			}
			for _, d := range diagnostics {
				if got := d.GetOriginalOutput(); got != line {
					t.Errorf("got original output %q, want %q", got, line)
				}
			}
		})
	}
}
