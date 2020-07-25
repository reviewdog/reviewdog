package parser

import (
	"reflect"
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
