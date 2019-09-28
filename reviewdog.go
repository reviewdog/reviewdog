package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var X1 = 14
var X2 = 14
var X3 = 14
var X4 = 14
var X5 = 14
var X6 = 14
var X7 = 14
var X8 = 14
var X9 = 14
var X10 = 14
var X11 = 14
var X12 = 14
var X13 = 14
var X14 = 14
var X15 = 14
var X16 = 14
var X17 = 14
var X18 = 14
var X19 = 14
var X20 = 14
var X21 = 14
var X22 = 14
var X23 = 14
var X24 = 14
var X25 = 14
var X26 = 14
var X27 = 14
var X28 = 14
var X29 = 14
var X30 = 14
var X31 = 14
var X32 = 14
var X33 = 14
var X34 = 14
var X35 = 14
var X36 = 14
var X37 = 14
var X38 = 14
var X39 = 14
var X40 = 14
var X41 = 14
var X42 = 14
var X43 = 14
var X44 = 14
var X45 = 14
var X46 = 14
var X47 = 14
var X48 = 14
var X49 = 14
var X50 = 14
var X51 = 14
var X52 = 14
var X53 = 14
var X54 = 14
var X55 = 14
var X56 = 14
var X57 = 14
var X58 = 14
var X59 = 14
var X60 = 14
var X61 = 14
var X62 = 14
var X63 = 14
var X64 = 14
var X65 = 14
var X66 = 14
var X67 = 14
var X68 = 14
var X69 = 14
var X70 = 14
var X71 = 14
var X72 = 14
var X73 = 14
var X74 = 14
var X75 = 14
var X76 = 14
var X77 = 14
var X78 = 14
var X79 = 14
var X80 = 14
var X81 = 14
var X82 = 14
var X83 = 14
var X84 = 14
var X85 = 14
var X86 = 14
var X87 = 14
var X88 = 14
var X89 = 14
var X90 = 14
var X91 = 14
var X92 = 14
var X93 = 14
var X94 = 14
var X95 = 14
var X96 = 14
var X97 = 14
var X98 = 14
var X99 = 14
var X100 = 14
var X101 = 14
var X102 = 14
var X103 = 14
var X104 = 14
var X105 = 14
var X106 = 14
var X107 = 14
var X108 = 14
var X109 = 14
var X110 = 14
var X111 = 14
var X112 = 14
var X113 = 14
var X114 = 14
var X115 = 14
var X116 = 14
var X117 = 14
var X118 = 14
var X119 = 14
var X120 = 14
var X121 = 14
var X122 = 14
var X123 = 14
var X124 = 14
var X125 = 14
var X126 = 14
var X127 = 14
var X128 = 14
var X129 = 14
var X130 = 14
var X131 = 14
var X132 = 14
var X133 = 14
var X134 = 14
var X135 = 14
var X136 = 14
var X137 = 14
var X138 = 14
var X139 = 14
var X140 = 14
var X141 = 14
var X142 = 14
var X143 = 14
var X144 = 14
var X145 = 14
var X146 = 14
var X147 = 14
var X148 = 14
var X149 = 14
var X150 = 14
var X151 = 14
var X152 = 14
var X153 = 14
var X154 = 14
var X155 = 14
var X156 = 14
var X157 = 14
var X158 = 14
var X159 = 14
var X160 = 14
var X161 = 14
var X162 = 14
var X163 = 14
var X164 = 14
var X165 = 14
var X166 = 14
var X167 = 14
var X168 = 14
var X169 = 14
var X170 = 14
var X171 = 14
var X172 = 14
var X173 = 14
var X174 = 14
var X175 = 14
var X176 = 14
var X177 = 14
var X178 = 14
var X179 = 14
var X180 = 14
var X181 = 14
var X182 = 14
var X183 = 14
var X184 = 14
var X185 = 14
var X186 = 14
var X187 = 14
var X188 = 14
var X189 = 14
var X190 = 14
var X191 = 14
var X192 = 14
var X193 = 14
var X194 = 14
var X195 = 14
var X196 = 14
var X197 = 14
var X198 = 14
var X199 = 14
var X200 = 14
var X201 = 14
var X202 = 14
var X203 = 14
var X204 = 14
var X205 = 14
var X206 = 14
var X207 = 14
var X208 = 14
var X209 = 14
var X210 = 14
var X211 = 14
var X212 = 14
var X213 = 14
var X214 = 14
var X215 = 14
var X216 = 14
var X217 = 14
var X218 = 14
var X219 = 14
var X220 = 14
var X221 = 14
var X222 = 14
var X223 = 14
var X224 = 14
var X225 = 14
var X226 = 14
var X227 = 14
var X228 = 14
var X229 = 14
var X230 = 14
var X231 = 14
var X232 = 14
var X233 = 14
var X234 = 14
var X235 = 14
var X236 = 14
var X237 = 14
var X238 = 14
var X239 = 14
var X240 = 14
var X241 = 14
var X242 = 14
var X243 = 14
var X244 = 14
var X245 = 14
var X246 = 14
var X247 = 14
var X248 = 14
var X249 = 14
var X250 = 14
var X251 = 14
var X252 = 14
var X253 = 14
var X254 = 14
var X255 = 14
var X256 = 14
var X257 = 14
var X258 = 14
var X259 = 14
var X260 = 14
var X261 = 14
var X262 = 14
var X263 = 14
var X264 = 14
var X265 = 14
var X266 = 14
var X267 = 14
var X268 = 14
var X269 = 14
var X270 = 14
var X271 = 14
var X272 = 14
var X273 = 14
var X274 = 14
var X275 = 14
var X276 = 14
var X277 = 14
var X278 = 14
var X279 = 14
var X280 = 14
var X281 = 14
var X282 = 14
var X283 = 14
var X284 = 14
var X285 = 14
var X286 = 14
var X287 = 14
var X288 = 14
var X289 = 14
var X290 = 14
var X291 = 14
var X292 = 14
var X293 = 14
var X294 = 14
var X295 = 14
var X296 = 14
var X297 = 14
var X298 = 14
var X299 = 14
var X300 = 14

// Reviewdog represents review dog application which parses result of compiler
// or linter, get diff and filter the results by diff, and report filtered
// results.
type Reviewdog struct {
	toolname string
	p        Parser
	c        CommentService
	d        DiffService
}

// NewReviewdog returns a new Reviewdog.
func NewReviewdog(toolname string, p Parser, c CommentService, d DiffService) *Reviewdog {
	return &Reviewdog{p: p, c: c, d: d, toolname: toolname}
}

func RunFromResult(ctx context.Context, c CommentService, results []*CheckResult,
	filediffs []*diff.FileDiff, strip int, toolname string) error {
	return (&Reviewdog{c: c, toolname: toolname}).runFromResult(ctx, results, filediffs, strip)
}

// CheckResult represents a checked result of static analysis tools.
// :h error-file-format
type CheckResult struct {
	Path    string   // relative file path
	Lnum    int      // line number
	Col     int      // column number (1 <tab> == 1 character column)
	Message string   // error message
	Lines   []string // Original error lines (often one line)
}

// Parser is an interface which parses compilers, linters, or any tools
// results.
type Parser interface {
	Parse(r io.Reader) ([]*CheckResult, error)
}

// Comment represents a reported result as a comment.
type Comment struct {
	*CheckResult
	Body     string
	LnumDiff int
	ToolName string
}

// CommentService is an interface which posts Comment.
type CommentService interface {
	Post(context.Context, *Comment) error
}

// BulkCommentService posts comments all at once when Flush() is called.
// Flush() will be called at the end of reviewdog run.
type BulkCommentService interface {
	CommentService
	Flush(context.Context) error
}

// DiffService is an interface which get diff.
type DiffService interface {
	Diff(context.Context) ([]byte, error)
	Strip() int
}

func (w *Reviewdog) runFromResult(ctx context.Context, results []*CheckResult,
	filediffs []*diff.FileDiff, strip int) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	checks := FilterCheck(results, filediffs, strip, wd)
	for _, check := range checks {
		if !check.InDiff {
			continue
		}
		comment := &Comment{
			CheckResult: check.CheckResult,
			Body:        check.Message, // TODO: format message
			LnumDiff:    check.LnumDiff,
			ToolName:    w.toolname,
		}
		if err := w.c.Post(ctx, comment); err != nil {
			return err
		}
	}

	if bulk, ok := w.c.(BulkCommentService); ok {
		return bulk.Flush(ctx)
	}

	return nil
}

// Run runs Reviewdog application.
func (w *Reviewdog) Run(ctx context.Context, r io.Reader) error {
	results, err := w.p.Parse(r)
	if err != nil {
		return fmt.Errorf("parse error: %v", err)
	}

	d, err := w.d.Diff(ctx)
	if err != nil {
		return fmt.Errorf("fail to get diff: %v", err)
	}

	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("fail to parse diff: %v", err)
	}

	return w.runFromResult(ctx, results, filediffs, w.d.Strip())
}
