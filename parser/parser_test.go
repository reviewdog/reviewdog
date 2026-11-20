package parser

import (
	"reflect"
	"strings"
	"testing"
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

func TestParsersSkipUTF8BOM(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	tests := []struct {
		name   string
		parser Parser
		in     string
		want   int
	}{
		{"rdjson", NewRDJSONParser(), bom + `{"diagnostics":[{"message":"m","location":{"path":"a.go"}}]}`, 1},
		{"rdjsonl", NewRDJSONLParser(), bom + `{"message":"m","location":{"path":"a.go"}}` + "\r\n", 1},
		{"sarif", NewSarifParser(), bom + `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"t"}},"results":[{"message":{"text":"m"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"a.go"},"region":{"startLine":1}}}]}]}]}`, 1},
		{"checkstyle", NewCheckStyleParser(), bom + `<?xml version="1.0"?><checkstyle><file name="a.go"><error line="1" message="m"/></file></checkstyle>`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, err := tt.parser.Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if len(ds) != tt.want {
				t.Fatalf("got %d diagnostics, want %d", len(ds), tt.want)
			}
			if got := ds[0].GetLocation().GetPath(); got != "a.go" {
				t.Errorf("got path %q, want a.go", got)
			}
			if got := ds[0].GetMessage(); got != "m" {
				t.Errorf("got message %q, want m", got)
			}
		})
	}
}
