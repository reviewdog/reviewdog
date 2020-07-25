package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		in      *ParserOpt
		typ     Parser
		wantErr bool
	}{
		{
			in: &ParserOpt{
				FormatName: "checkstyle",
			},
			typ: &CheckStyleParser{},
		},
		{
			in: &ParserOpt{
				FormatName: "rdjsonl",
			},
			typ: &RDJSONLParser{},
		},
		{
			in: &ParserOpt{
				FormatName: "golint",
			},
			typ: &ErrorformatParser{},
		},
		{
			in: &ParserOpt{
				Errorformat: []string{`%f:%l:%c:%m`},
			},
			typ: &ErrorformatParser{},
		},
		{ // empty
			in:      &ParserOpt{},
			wantErr: true,
		},
		{ // both
			in: &ParserOpt{
				FormatName:  "checkstyle",
				Errorformat: []string{`%f:%l:%c:%m`},
			},
			wantErr: true,
		},
		{ // unsupported
			in: &ParserOpt{
				FormatName: "unsupported format",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		p, err := NewParser(tt.in)
		if tt.wantErr && err != nil {
			continue
		}
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := reflect.TypeOf(p), reflect.TypeOf(tt.typ); got != want {
			t.Errorf("typ: got %v, want %v", got, want)
		}
	}
}

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

func TestCheckStyleParser(t *testing.T) {
	const sample = `<?xml version="1.0" encoding="utf-8"?><checkstyle version="4.3"><file name="/path/to/file"><error line="1" column="10" severity="error" message="&apos;addOne&apos; is defined but never used. (no-unused-vars)" source="eslint.rules.no-unused-vars" /><error line="2" column="9" severity="error" message="Use the isNaN function to compare with NaN. (use-isnan)" source="eslint.rules.use-isnan" /><error line="3" column="16" severity="error" message="Unexpected space before unary operator &apos;++&apos;. (space-unary-ops)" source="eslint.rules.space-unary-ops" /><error line="3" column="20" severity="warning" message="Missing semicolon. (semi)" source="eslint.rules.semi" /><error line="4" column="12" severity="warning" message="Unnecessary &apos;else&apos; after &apos;return&apos;. (no-else-return)" source="eslint.rules.no-else-return" /><error line="5" column="7" severity="warning" message="Expected indentation of 8 spaces but found 6. (indent)" source="eslint.rules.indent" /><error line="5" column="7" severity="error" message="Expected a return value. (consistent-return)" source="eslint.rules.consistent-return" /><error line="5" column="13" severity="warning" message="Missing semicolon. (semi)" source="eslint.rules.semi" /><error line="7" column="2" severity="error" message="Unnecessary semicolon. (no-extra-semi)" source="eslint.rules.no-extra-semi" /></file></checkstyle>`

	wants := []string{

		"/path/to/file:1:10: error: 'addOne' is defined but never used. (no-unused-vars) (eslint.rules.no-unused-vars)",
		"/path/to/file:2:9: error: Use the isNaN function to compare with NaN. (use-isnan) (eslint.rules.use-isnan)",
		"/path/to/file:3:16: error: Unexpected space before unary operator '++'. (space-unary-ops) (eslint.rules.space-unary-ops)",
		"/path/to/file:3:20: warning: Missing semicolon. (semi) (eslint.rules.semi)",
		"/path/to/file:4:12: warning: Unnecessary 'else' after 'return'. (no-else-return) (eslint.rules.no-else-return)",
		"/path/to/file:5:7: warning: Expected indentation of 8 spaces but found 6. (indent) (eslint.rules.indent)",
		"/path/to/file:5:7: error: Expected a return value. (consistent-return) (eslint.rules.consistent-return)",
		"/path/to/file:5:13: warning: Missing semicolon. (semi) (eslint.rules.semi)",
		"/path/to/file:7:2: error: Unnecessary semicolon. (no-extra-semi) (eslint.rules.no-extra-semi)",
	}

	p := NewCheckStyleParser()
	diagnostics, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		t.Error(err)
	}
	for i, d := range diagnostics {
		if got, want := d.GetOriginalOutput(), wants[i]; got != want {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestRDJSONLParser(t *testing.T) {
	const sample = `{"source":{"name":"deadcode"},"message":"'unused' is unused","location":{"path":"testdata/main.go","range":{"start":{"line":18,"column":6}}}}
{"source":{"name":"deadcode"},"message":"'unused2' is unused","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}}
{"source":{"name":"errcheck"},"message":"Error return value of 'os.Open' is not checked","location":{"path":"testdata/main.go","range":{"start":{"line":15,"column":9}}}}
{"source":{"name":"ineffassign"},"message":"ineffectual assignment to 'x'","location":{"path":"testdata/main.go","range":{"start":{"line":12,"column":2}}}}
{"source":{"name":"govet"},"message":"printf: Sprintf format %d reads arg #1, but call has 0 args","location":{"path":"testdata/main.go","range":{"start":{"line":13,"column":2}}}}
{"source":{"name":"severity-test"},"message":"severity test (string)","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}, "severity": "WARNING"}
{"source":{"name":"severity-test"},"message":"severity test (number)","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}, "severity": "WARNING"}`
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
