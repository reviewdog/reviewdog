package parser

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/reviewdog/errorformat"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

var _ Parser = &ErrorformatParser{}

// ErrorformatParser is errorformat parser.
type ErrorformatParser struct {
	efm *errorformat.Errorformat
}

// NewErrorformatParser returns a new ErrorformatParser.
func NewErrorformatParser(efm *errorformat.Errorformat) *ErrorformatParser {
	return &ErrorformatParser{efm: efm}
}

// NewErrorformatParserString returns a new ErrorformatParser from errorformat
// in string representation.
func NewErrorformatParserString(efms []string) (*ErrorformatParser, error) {
	efm, err := errorformat.NewErrorformat(efms)
	if err != nil {
		return nil, err
	}
	return NewErrorformatParser(efm), nil
}

// maxLineLen is the longest line, in bytes without the line break, that
// ErrorformatParser hands to the errorformat scanner. The scanner reads lines
// with a bufio.Scanner, which stops at a line it cannot hold in
// bufio.MaxScanTokenSize bytes and reports no error, so the rest of the input
// would be dropped without a trace.
const maxLineLen = bufio.MaxScanTokenSize - 2

// lineLimitReader cuts every line of r to max bytes and drops the rest of that
// line. The line breaks themselves are kept. A multi-byte UTF-8 character that
// would cross the limit is dropped whole, so the cut never leaves half of one.
type lineLimitReader struct {
	r         io.Reader
	max       int
	n         int  // bytes passed through in the current line
	truncated bool // some bytes were dropped
}

func (l *lineLimitReader) Read(p []byte) (int, error) {
	for {
		n, err := l.r.Read(p)
		w := 0 // w never passes the byte being read, so p is compacted in place.
		for _, b := range p[:n] {
			if b == '\n' {
				l.n = 0
			} else if l.n+seqLen(b) > l.max {
				l.n = l.max // the rest of the line is dropped too
				l.truncated = true
				continue
			} else {
				l.n++
			}
			p[w] = b
			w++
		}
		// Read again rather than return (0, nil) when the whole chunk was dropped.
		if w > 0 || err != nil || n == 0 {
			return w, err
		}
	}
}

// seqLen returns the length of the UTF-8 sequence that b starts, and 1 for any
// other byte.
func seqLen(b byte) int {
	switch {
	case b >= 0xF0:
		return 4
	case b >= 0xE0:
		return 3
	case b >= 0xC0:
		return 2
	}
	return 1
}

func (p *ErrorformatParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	lr := &lineLimitReader{r: r, max: maxLineLen}
	s := p.efm.NewScanner(lr)
	var ds []*rdf.Diagnostic
	for s.Scan() {
		e := s.Entry()
		if e.Valid {
			d := &rdf.Diagnostic{
				Location: &rdf.Location{
					Path: e.Filename,
					Range: &rdf.Range{
						Start: &rdf.Position{
							Line:   int32(e.Lnum),
							Column: int32(e.Col),
						},
					},
				},
				Message:        e.Text,
				Severity:       severity(string(e.Type)),
				OriginalOutput: strings.Join(e.Lines, "\n"),
			}
			if e.Nr != 0 {
				d.Code = &rdf.Code{Value: fmt.Sprintf("%d", e.Nr)}
			}
			ds = append(ds, d)
		}
	}
	if lr.truncated {
		slog.Warn("reviewdog: cut input lines longer than the limit before parsing", "limit", maxLineLen)
	}
	return ds, nil
}
