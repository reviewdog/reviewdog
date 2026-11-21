package parser

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"unicode/utf16"

	"golang.org/x/text/transform"
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

// encodeUTF16 encodes s as UTF-16 with a byte order mark.
func encodeUTF16(s string, bigEndian bool) string {
	var b []byte
	if bigEndian {
		b = append(b, 0xFE, 0xFF)
	} else {
		b = append(b, 0xFF, 0xFE)
	}
	for _, u := range utf16.Encode([]rune(s)) {
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
		{"UTF-16LE BOM", func(s string) string { return encodeUTF16(s, false) }},
		{"UTF-16BE BOM", func(s string) string { return encodeUTF16(s, true) }},
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

func TestCheckStyleParserBOMOverridesDeclaration(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="%s"?><checkstyle><file name="a.go"><error line="1" message="café"/></file></checkstyle>`
	tests := []struct {
		name string
		in   string
	}{
		{"UTF-16LE BOM, UTF-16", encodeUTF16(fmt.Sprintf(doc, "UTF-16"), false)},
		{"UTF-16LE BOM, unicode alias", encodeUTF16(fmt.Sprintf(doc, "unicode"), false)},
		{"UTF-16BE BOM, unicodefffe alias", encodeUTF16(fmt.Sprintf(doc, "unicodefffe"), true)},
		{"UTF-16LE BOM, UTF-16BE declared", encodeUTF16(fmt.Sprintf(doc, "UTF-16BE"), false)},
		{"UTF-8 BOM, windows-1252 declared", "\xef\xbb\xbf" + fmt.Sprintf(doc, "windows-1252")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, err := NewCheckStyleParser().Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if len(ds) != 1 || ds[0].GetMessage() != "caf\xc3\xa9" {
				t.Fatalf("got %v, want one diagnostic with message caf\xc3\xa9", ds)
			}
		})
	}
}

func TestErrorformatParserBOMInMiddle(t *testing.T) {
	// e.g. project mode, where stderr is appended to stdout and both start with a BOM
	const bom = "\xef\xbb\xbf"
	tests := []struct {
		efm  string
		line func(path string) string
	}{
		{`%f:%l:%c: %m`, func(path string) string { return path + ":1:2: m" }},
		{`ERROR: %f:%l: %m`, func(path string) string { return "ERROR: " + path + ":1: m" }},
	}
	for _, tt := range tests {
		t.Run(tt.efm, func(t *testing.T) {
			efm, err := NewErrorformatParserString([]string{tt.efm})
			if err != nil {
				t.Fatal(err)
			}
			in := bom + tt.line("a.go") + "\n" + bom + tt.line("b.go") + "\r\n" + bom + tt.line("c.go")
			ds, err := efm.Parse(strings.NewReader(in))
			if err != nil {
				t.Fatal(err)
			}
			if len(ds) != 3 {
				t.Fatalf("got %d diagnostics, want 3", len(ds))
			}
			for i, want := range []string{"a.go", "b.go", "c.go"} {
				if got := ds[i].GetLocation().GetPath(); got != want {
					t.Errorf("diagnostic %d: got path %q, want %q", i, got, want)
				}
				if got := ds[i].GetOriginalOutput(); strings.HasPrefix(got, bom) {
					t.Errorf("diagnostic %d: original output %q starts with a BOM", i, got)
				}
			}
		})
	}
}

func TestLineBOMStripperChunkBoundaries(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	in := bom + "a\n" + bom + "b\n\n" + "\xef\xbb" + "c\n" + bom
	want := "a\nb\n\n\xef\xbbc\n"
	// one byte at a time, so every BOM is split across reads
	got, err := io.ReadAll(transform.NewReader(iotest.OneByteReader(strings.NewReader(in)), &lineBOMStripper{}))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeBOMKeepsReadError(t *testing.T) {
	errBoom := errors.New("boom")
	for _, in := range []string{"a", "\xef\xbb\xbf", "\xff\xfe"} {
		r := &onceErrReader{data: []byte(in), err: errBoom}
		if _, err := io.ReadAll(decodeBOM(r)); !errors.Is(err, errBoom) {
			t.Errorf("%q: got error %v, want %v", in, err, errBoom)
		}
	}
}

// onceErrReader returns data, then err once, then io.EOF.
type onceErrReader struct {
	data []byte
	err  error
}

func (r *onceErrReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	if r.err != nil {
		err := r.err
		r.err = nil
		return 0, err
	}
	return 0, io.EOF
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
	efm, err := NewErrorformatParserString([]string{`%f:%l:%c: %m`})
	if err != nil {
		t.Fatal(err)
	}
	parsers := []struct {
		name   string
		parser Parser
	}{
		{"rdjson", NewRDJSONParser()},
		{"rdjsonl", NewRDJSONLParser()},
		{"sarif", NewSarifParser()},
		{"checkstyle", NewCheckStyleParser()},
		{"errorformat", efm},
		{"diff", NewDiffParser(0)},
	}
	for _, p := range parsers {
		// A BOM alone must behave exactly like empty input.
		wantDs, wantErr := p.parser.Parse(strings.NewReader(""))
		for _, in := range []string{"\xef\xbb\xbf", "\xff\xfe", "\xfe\xff"} {
			ds, err := p.parser.Parse(strings.NewReader(in))
			if (err != nil) != (wantErr != nil) || len(ds) != len(wantDs) {
				t.Errorf("%s %q: got (%d, %v), want (%d, %v) as for empty input", p.name, in, len(ds), err, len(wantDs), wantErr)
			}
		}
		// Inputs shorter than a UTF-8 BOM must not panic.
		for _, in := range []string{"a", "\xef", "\xef\xbb"} {
			_, _ = p.parser.Parse(strings.NewReader(in))
		}
	}
}
