package parser

import (
	"reflect"
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
