package parser

import (
	"bufio"
	"bytes"
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
	DiffStrip   int
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
	case "rdjson":
		return NewRDJSONParser(), nil
	case "diff":
		return NewDiffParser(opt.DiffStrip), nil
	case "sarif":
		return NewSarifParser(), nil
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

func severity(s string) rdf.Severity {
	switch s {
	case "error", "ERROR", "Error", "e", "E":
		return rdf.Severity_ERROR
	case "warning", "WARNING", "Warning", "w", "W":
		return rdf.Severity_WARNING
	case "info", "INFO", "Info", "i", "I",
		"note", "NOTE", "Note", "n", "N": // Treat note as info.
		return rdf.Severity_INFO
	default:
		return rdf.Severity_UNKNOWN_SEVERITY
	}
}

// utf8BOM is the UTF-8 byte order mark. Tools on Windows (PowerShell's
// Out-File, Notepad, some .NET writers) put it in front of JSON files, but JSON
// parsers reject it as an invalid first character.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// skipUTF8BOM returns a reader that yields r without a leading UTF-8 byte
// order mark.
func skipUTF8BOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	if b, err := br.Peek(len(utf8BOM)); err == nil && bytes.Equal(b, utf8BOM) {
		_, _ = br.Discard(len(utf8BOM))
	}
	return br
}
