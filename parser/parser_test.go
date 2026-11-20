package parser

import (
	"reflect"
	"strings"
	"testing"
	utf16enc "unicode/utf16"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		in      *Option
		typ     Parser
		wantErr bool
	}{
		{
			in: &Option{
				FormatName: "checkstyle",
			},
			typ: &CheckStyleParser{},
		},
		{
			in: &Option{
				FormatName: "rdjsonl",
			},
			typ: &RDJSONLParser{},
		},
		{
			in: &Option{
				FormatName: "golint",
			},
			typ: &ErrorformatParser{},
		},
		{
			in: &Option{
				Errorformat: []string{`%f:%l:%c:%m`},
			},
			typ: &ErrorformatParser{},
		},
		{
			in: &Option{
				FormatName: "sarif",
			},
			typ: &SarifParser{},
		},
		{ // empty
			in:      &Option{},
			wantErr: true,
		},
		{ // both
			in: &Option{
				FormatName:  "checkstyle",
				Errorformat: []string{`%f:%l:%c:%m`},
			},
			wantErr: true,
		},
		{ // unsupported
			in: &Option{
				FormatName: "unsupported format",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		p, err := New(tt.in)
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

// utf16 encodes s as UTF-16 with a byte order mark.
func utf16(s string, bigEndian bool) string {
	var b []byte
	if bigEndian {
		b = append(b, 0xFE, 0xFF)
	} else {
		b = append(b, 0xFF, 0xFE)
	}
	for _, u := range utf16enc.Encode([]rune(s)) {
		if bigEndian {
			b = append(b, byte(u>>8), byte(u))
		} else {
			b = append(b, byte(u), byte(u>>8))
		}
	}
	return string(b)
}

func TestParsersByteOrderMark(t *testing.T) {
	const utf8BOM = "\xef\xbb\xbf"
	efm, err := NewErrorformatParserString([]string{`%f:%l:%c: %m`})
	if err != nil {
		t.Fatal(err)
	}
	inputs := []struct {
		name   string
		parser Parser
		in     string
	}{
		{"rdjson", NewRDJSONParser(), `{"diagnostics":[{"message":"m","location":{"path":"a.go"}}]}`},
		{"rdjsonl", NewRDJSONLParser(), `{"message":"m","location":{"path":"a.go"}}` + "\r\n"},
		{"sarif", NewSarifParser(), `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"t"}},"results":[{"message":{"text":"m"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"a.go"},"region":{"startLine":1}}}]}]}]}`},
		{"checkstyle", NewCheckStyleParser(), `<?xml version="1.0"?><checkstyle><file name="a.go"><error line="1" message="m"/></file></checkstyle>`},
		{"errorformat", efm, "a.go:1:2: m\n"},
		{"diff", NewDiffParser(0), "--- a.go\n+++ a.go\n@@ -1 +1 @@\n-x\n+m\n"},
	}
	encodings := []struct {
		name   string
		encode func(string) string
	}{
		{"UTF-8 BOM", func(s string) string { return utf8BOM + s }},
		{"UTF-16LE BOM", func(s string) string { return utf16(s, false) }},
		{"UTF-16BE BOM", func(s string) string { return utf16(s, true) }},
	}
	for _, in := range inputs {
		for _, enc := range encodings {
			t.Run(in.name+"/"+enc.name, func(t *testing.T) {
				ds, err := in.parser.Parse(strings.NewReader(enc.encode(in.in)))
				if err != nil {
					t.Fatal(err)
				}
				if len(ds) != 1 {
					t.Fatalf("got %d diagnostics, want 1", len(ds))
				}
				if got := ds[0].GetLocation().GetPath(); got != "a.go" {
					t.Errorf("got path %q, want a.go", got)
				}
				if in.name != "diff" {
					if got := ds[0].GetMessage(); got != "m" {
						t.Errorf("got message %q, want m", got)
					}
				}
				if got := ds[0].GetOriginalOutput(); strings.HasPrefix(got, "\ufeff") {
					t.Errorf("original output %q starts with a BOM", got)
				}
			})
		}
	}
}

func TestCheckStyleParserUTF16Declaration(t *testing.T) {
	in := utf16(`<?xml version="1.0" encoding="UTF-16"?><checkstyle><file name="a.go"><error line="1" message="m"/></file></checkstyle>`, false)
	ds, err := NewCheckStyleParser().Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].GetMessage() != "m" {
		t.Fatalf("got %v, want one diagnostic with message m", ds)
	}
}

func TestRDJSONLParserBOMPerLine(t *testing.T) {
	// e.g. cat a.jsonl b.jsonl where both files start with a BOM
	const bom = "\xef\xbb\xbf"
	in := bom + `{"message":"a","location":{"path":"a.go"}}` + "\n" + bom + `{"message":"b","location":{"path":"b.go"}}` + "\n"
	ds, err := NewRDJSONLParser().Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 {
		t.Fatalf("got %d diagnostics, want 2", len(ds))
	}
	for _, d := range ds {
		if strings.HasPrefix(d.GetOriginalOutput(), "\ufeff") {
			t.Errorf("original output %q starts with a BOM", d.GetOriginalOutput())
		}
	}
}

func TestParsersShortInput(t *testing.T) {
	for _, in := range []string{"", "\xef\xbb\xbf", "\xff\xfe", "\n", "{}"} {
		ds, err := NewRDJSONLParser().Parse(strings.NewReader(in))
		if err != nil {
			t.Errorf("rdjsonl %q: %v", in, err)
		}
		want := 0
		if in == "{}" {
			want = 1
		}
		if len(ds) != want {
			t.Errorf("rdjsonl %q: got %d diagnostics, want %d", in, len(ds), want)
		}
	}
	efm, err := NewErrorformatParserString([]string{`%f:%l:%c: %m`})
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{"", "\xef\xbb\xbf", "\xfe\xff", "a"} {
		if _, err := efm.Parse(strings.NewReader(in)); err != nil {
			t.Errorf("errorformat %q: %v", in, err)
		}
	}
}
