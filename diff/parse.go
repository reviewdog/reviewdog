package diff

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	tokenDiffGit        = "diff --git" // diff --git a/sample.orig.txt b/sample.new.txt
	tokenOrigFile       = "---"        // --- sample.orig.txt	2016-10-13 05:09:35.820791185 +0900
	tokenNewFile        = "+++"        // +++ sample.new.txt	2016-10-13 05:15:26.839245048 +0900
	tokenStartHunk      = "@@"         // @@ -1,3 +1,4 @@
	tokenUnchangedLine  = " "          //  unchanged, contextual line
	tokenAddedLine      = "+"          // +added line
	tokenDeletedLine    = "-"          // -deleted line
	tokenNoNewlineAtEOF = `\`          // \ No newline at end of file
)

var (
	// ErrNoNewFile represents error which there are no expected new file line.
	ErrNoNewFile = errors.New("no expected new file line") // +++ newfile
	// ErrNoHunks represents error which there are no expected hunks.
	ErrNoHunks = errors.New("no expected hunks") // @@ -1,3 +1,4 @@
)

// ErrInvalidHunkRange represents invalid line of hunk range. @@ -1,3 +1,4 @@
type ErrInvalidHunkRange struct {
	invalid string
}

func (e *ErrInvalidHunkRange) Error() string {
	return fmt.Sprintf("invalid hunk range: %v", e.invalid)
}

// ParseMultiFile parses a multi-file unified diff.
func ParseMultiFile(r io.Reader) ([]*FileDiff, error) {
	return (&multiFileParser{r: bufio.NewReader(r)}).Parse()
}

type multiFileParser struct {
	r *bufio.Reader
}

func (p *multiFileParser) Parse() ([]*FileDiff, error) {
	var fds []*FileDiff
	fp := &fileParser{r: p.r}
	for {
		fd, err := fp.Parse()
		if err != nil || fd == nil {
			break
		}
		fds = append(fds, fd)
	}
	return fds, nil
}

// ParseFile parses a file unified diff.
func ParseFile(r io.Reader) (*FileDiff, error) {
	return (&fileParser{r: bufio.NewReader(r)}).Parse()
}

type fileParser struct {
	r *bufio.Reader
}

func (p *fileParser) Parse() (*FileDiff, error) {
	fd := &FileDiff{}
	fd.Extended = parseExtenedHeader(p.r)
	b, err := p.r.Peek(len(tokenOrigFile))
	if err != nil {
		if err == io.EOF && len(fd.Extended) > 0 {
			return fd, nil
		}
		return nil, nil
	}
	if bytes.HasPrefix(b, []byte(tokenOrigFile)) {
		// parse `--- sample.orig.txt	2016-10-13 05:09:35.820791185 +0900`
		origline, _ := readline(p.r) // ignore err because we know it can read something
		fd.NameOrig, fd.TimeOrig = parseFileHeader(origline)
		// parse `+++ sample.new.txt	2016-10-13 05:09:35.820791185 +0900`
		if b, err := p.r.Peek(len(tokenNewFile)); err != nil || !bytes.HasPrefix(b, []byte(tokenNewFile)) {
			return nil, ErrNoNewFile
		}
		newline, _ := readline(p.r) // ignore err because we know it can read something
		fd.NameNew, fd.TimeNew = parseFileHeader(newline)
	}
	// parse hunks
	fd.Hunks, err = p.parseHunks()
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (p *fileParser) parseHunks() ([]*Hunk, error) {
	b, err := p.r.Peek(len(tokenOrigFile))
	if err != nil {
		return nil, ErrNoHunks
	}
	if !bytes.HasPrefix(b, []byte(tokenStartHunk)) {
		return nil, ErrNoHunks
	}
	var hunks []*Hunk
	hp := &hunkParser{r: p.r}
	for {
		h, err := hp.Parse()
		if err != nil {
			return nil, err
		}
		if h == nil {
			break
		}
		hunks = append(hunks, h)
	}
	return hunks, nil
}

// parseFileHeader parses file header line and returns filename and timestamp.
// timestamp may be empty.
func parseFileHeader(line string) (filename, timestamp string) {
	// strip `+++ ` or `--- `
	ss := line[len(tokenOrigFile)+1:]
	tabi := strings.LastIndex(ss, "\t")
	if tabi == -1 {
		return ss, ""
	}
	return ss[:tabi], ss[tabi+1:]
}

func parseExtenedHeader(r *bufio.Reader) []string {
	var es []string
	b, err := r.Peek(len(tokenDiffGit))
	if err != nil {
		return nil
	}
	// if starts with 'diff --git', parse extended header
	if bytes.HasPrefix(b, []byte(tokenDiffGit)) {
		diffgitline, _ := readline(r) // ignore err because we know it can read something
		es = append(es, diffgitline)
		for {
			b, err := r.Peek(len(tokenDiffGit))
			if err != nil || bytes.HasPrefix(b, []byte(tokenOrigFile)) || bytes.HasPrefix(b, []byte(tokenDiffGit)) {
				break
			}
			line, _ := readline(r)
			es = append(es, string(line))
		}
	}
	return es
}

type hunkParser struct {
	r        *bufio.Reader
	lnumdiff int
}

func (p *hunkParser) Parse() (*Hunk, error) {
	if b, err := p.r.Peek(len(tokenStartHunk)); err != nil || !bytes.HasPrefix(b, []byte(tokenStartHunk)) {
		return nil, nil
	}
	rangeline, _ := readline(p.r)
	hr, err := parseHunkRange(rangeline)
	if err != nil {
		return nil, err
	}
	hunk := &Hunk{
		StartLineOrig:  hr.lorig,
		LineLengthOrig: hr.sorig,
		StartLineNew:   hr.lnew,
		LineLengthNew:  hr.snew,
		Section:        hr.section,
	}
	lorig := hr.lorig
	lnew := hr.lnew
	for {
		b, err := p.r.Peek(1)
		if err != nil {
			break
		}
		token := string(b)
		switch token {
		case tokenUnchangedLine, tokenAddedLine, tokenDeletedLine:
			p.lnumdiff++
			l, _ := readline(p.r)
			line := &Line{Content: l[len(token):]} // trim first token
			switch token {
			case tokenUnchangedLine:
				line.Type = LineUnchanged
				line.LnumDiff = p.lnumdiff
				line.LnumOrig = lorig
				line.LnumNew = lnew
				lorig++
				lnew++
			case tokenAddedLine:
				line.Type = LineAdded
				line.LnumDiff = p.lnumdiff
				line.LnumNew = lnew
				lnew++
			case tokenDeletedLine:
				line.Type = LineAdded
				line.LnumDiff = p.lnumdiff
				line.LnumOrig = lorig
				lorig++
			}
			hunk.Lines = append(hunk.Lines, line)
		case tokenNoNewlineAtEOF:
			// skip \ No newline at end of file. just consume line
			readline(p.r)
		default:
			return hunk, nil
		}
	}
	return hunk, nil
}

// @@ -l,s +l,s @@ optional section heading
type hunkrange struct {
	lorig, sorig, lnew, snew int
	section                  string
}

// @@ -lorig[,sorig] +lnew[,snew] @@[ section]
// 0  1              2            3   4
func parseHunkRange(rangeline string) (*hunkrange, error) {
	ps := strings.SplitN(rangeline, " ", 5)
	invalidErr := &ErrInvalidHunkRange{invalid: rangeline}
	hunkrange := &hunkrange{}
	if len(ps) < 4 || ps[0] != "@@" || ps[3] != "@@" {
		return nil, invalidErr
	}
	orig := ps[1] // -lorig[,sorig]
	if !strings.HasPrefix(orig, "-") {
		return nil, invalidErr
	}
	lorig, sorig, err := parseLS(orig[1:])
	if err != nil {
		return nil, invalidErr
	}
	hunkrange.lorig = lorig
	hunkrange.sorig = sorig
	new := ps[2] // +lnew[,snew]
	if !strings.HasPrefix(new, "+") {
		return nil, invalidErr
	}
	lnew, snew, err := parseLS(new[1:])
	if err != nil {
		return nil, invalidErr
	}
	hunkrange.lnew = lnew
	hunkrange.snew = snew
	if len(ps) == 5 {
		hunkrange.section = ps[4]
	}
	return hunkrange, nil
}

// l[,s]
func parseLS(ls string) (l, s int, err error) {
	ss := strings.SplitN(ls, ",", 2)
	l, err = strconv.Atoi(ss[0])
	if err != nil {
		return 0, 0, err
	}
	if len(ss) == 2 {
		s, err = strconv.Atoi(ss[1])
		if err != nil {
			return 0, 0, err
		}
	} else {
		s = 1
	}
	return l, s, nil
}

// readline reads lines from bufio.Reader with size limit. It consumes
// remaining content even if the line size reaches size limit.
func readline(r *bufio.Reader) (string, error) {
	line, isPrefix, err := r.ReadLine()
	if err != nil {
		return "", err
	}
	// consume all remaining line content
	for isPrefix {
		_, isPrefix, _ = r.ReadLine()
	}
	return string(line), nil
}
