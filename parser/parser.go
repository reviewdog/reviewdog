package parser

import (
	"errors"
	"fmt"
	"io"

	"github.com/reviewdog/errorformat/fmts"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

// Parser is an interface which parses compilers, linters, or any tools
// results.
type Parser interface {
	Parse(r io.Reader) ([]*rdf.Diagnostic, error)
}

// Option represents option to create Parser. Either FormatName or
// Errorformat should be specified.
type Option struct {
	FormatName  string
	Errorformat []string
}

// New returns Parser based on Option.
func New(opt *Option) (Parser, error) {
	name := opt.FormatName

	if name != "" && len(opt.Errorformat) > 0 {
		return nil, errors.New("you cannot specify both format name and errorformat at the same time")
	}

	switch name {
	case "checkstyle":
		return NewCheckStyleParser(), nil
	case "rdjsonl":
		return NewRDJSONLParser(), nil
	}

	// use defined errorformat
	if name != "" {
		efm, ok := fmts.DefinedFmts()[name]
		if !ok {
			return nil, fmt.Errorf("%q is not supported. consider to add new errorformat to https://github.com/reviewdog/errorformat", name)
		}
		opt.Errorformat = efm.Errorformat
	}
	if len(opt.Errorformat) == 0 {
		return nil, errors.New("errorformat is empty")
	}
	return NewErrorformatParserString(opt.Errorformat)
}
