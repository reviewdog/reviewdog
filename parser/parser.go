package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

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

// decodeBOM returns a reader that drops a leading byte order mark from r and,
// for a UTF-16 one, decodes the rest to UTF-8. Tools on Windows write these:
// Windows PowerShell 5.1's > and Out-File produce UTF-16LE with a BOM, and
// -Encoding utf8 there (utf8BOM in PowerShell 7) or some .NET writers UTF-8
// with a BOM. Input without a BOM is passed through unchanged.
func decodeBOM(r io.Reader) io.Reader {
	rd, _ := stripBOM(r)
	return rd
}

// stripBOM is decodeBOM that also reports whether r started with a BOM. Only
// input that has one goes through the transformer.
func stripBOM(r io.Reader) (io.Reader, bool) {
	br := bufio.NewReader(r)
	b, err := br.Peek(3)
	var rd io.Reader = br
	if err != nil {
		// Peek clears the error from br, so keep it for the caller instead
		// of reading r again after it has failed or ended.
		rd = io.MultiReader(bytes.NewReader(b), errReader{err})
	}
	if !bytes.HasPrefix(b, utf8BOM) && !bytes.HasPrefix(b, utf16LEBOM) && !bytes.HasPrefix(b, utf16BEBOM) {
		return rd, false
	}
	return transform.NewReader(rd, unicode.BOMOverride(transform.Nop)), true
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// lineBOMStripper is a transform.Transformer that drops a UTF-8 BOM at the
// start of every line, e.g. where concatenated files or stderr appended to
// stdout begin.
type lineBOMStripper struct{ midLine bool }

func (t *lineBOMStripper) Reset() { t.midLine = false }

func (t *lineBOMStripper) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		if !t.midLine {
			rest := src[nSrc:]
			if !atEOF && len(rest) < len(utf8BOM) && bytes.HasPrefix(utf8BOM, rest) {
				return nDst, nSrc, transform.ErrShortSrc
			}
			if bytes.HasPrefix(rest, utf8BOM) {
				nSrc += len(utf8BOM)
			}
			t.midLine = true
			continue
		}
		if nDst == len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		c := src[nSrc]
		dst[nDst] = c
		nDst++
		nSrc++
		if c == '\n' {
			t.midLine = false
		}
	}
	return nDst, nSrc, nil
}

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
)
