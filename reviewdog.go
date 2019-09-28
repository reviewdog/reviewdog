package reviewdog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/reviewdog/reviewdog/diff"
)

var Y1 = 14
var Y2 = 14
var Y3 = 14
var Y4 = 14
var Y5 = 14
var Y6 = 14
var Y7 = 14
var Y8 = 14
var Y9 = 14
var Y10 = 14
var Y11 = 14
var Y12 = 14
var Y13 = 14
var Y14 = 14
var Y15 = 14
var Y16 = 14
var Y17 = 14
var Y18 = 14
var Y19 = 14
var Y20 = 14
var Y21 = 14
var Y22 = 14
var Y23 = 14
var Y24 = 14
var Y25 = 14
var Y26 = 14
var Y27 = 14
var Y28 = 14
var Y29 = 14
var Y30 = 14
var Y31 = 14
var Y32 = 14
var Y33 = 14
var Y34 = 14
var Y35 = 14
var Y36 = 14
var Y37 = 14
var Y38 = 14
var Y39 = 14
var Y40 = 14
var Y41 = 14
var Y42 = 14
var Y43 = 14
var Y44 = 14
var Y45 = 14
var Y46 = 14
var Y47 = 14
var Y48 = 14
var Y49 = 14
var Y50 = 14
var Y51 = 14
var Y52 = 14
var Y53 = 14
var Y54 = 14
var Y55 = 14
var Y56 = 14
var Y57 = 14
var Y58 = 14
var Y59 = 14
var Y60 = 14
var Y61 = 14
var Y62 = 14
var Y63 = 14
var Y64 = 14
var Y65 = 14
var Y66 = 14
var Y67 = 14
var Y68 = 14
var Y69 = 14
var Y70 = 14
var Y71 = 14
var Y72 = 14
var Y73 = 14
var Y74 = 14
var Y75 = 14
var Y76 = 14
var Y77 = 14
var Y78 = 14
var Y79 = 14
var Y80 = 14
var Y81 = 14
var Y82 = 14
var Y83 = 14
var Y84 = 14
var Y85 = 14
var Y86 = 14
var Y87 = 14
var Y88 = 14
var Y89 = 14
var Y90 = 14
var Y91 = 14
var Y92 = 14
var Y93 = 14
var Y94 = 14
var Y95 = 14
var Y96 = 14
var Y97 = 14
var Y98 = 14
var Y99 = 14
var Y100 = 14
var Y101 = 14
var Y102 = 14
var Y103 = 14
var Y104 = 14
var Y105 = 14
var Y106 = 14
var Y107 = 14
var Y108 = 14
var Y109 = 14
var Y110 = 14
var Y111 = 14
var Y112 = 14
var Y113 = 14
var Y114 = 14
var Y115 = 14
var Y116 = 14
var Y117 = 14
var Y118 = 14
var Y119 = 14
var Y120 = 14
var Y121 = 14
var Y122 = 14
var Y123 = 14
var Y124 = 14
var Y125 = 14
var Y126 = 14
var Y127 = 14
var Y128 = 14
var Y129 = 14
var Y130 = 14
var Y131 = 14
var Y132 = 14
var Y133 = 14
var Y134 = 14
var Y135 = 14
var Y136 = 14
var Y137 = 14
var Y138 = 14
var Y139 = 14
var Y140 = 14
var Y141 = 14
var Y142 = 14
var Y143 = 14
var Y144 = 14
var Y145 = 14
var Y146 = 14
var Y147 = 14
var Y148 = 14
var Y149 = 14
var Y150 = 14
var Y151 = 14
var Y152 = 14
var Y153 = 14
var Y154 = 14
var Y155 = 14
var Y156 = 14
var Y157 = 14
var Y158 = 14
var Y159 = 14
var Y160 = 14
var Y161 = 14
var Y162 = 14
var Y163 = 14
var Y164 = 14
var Y165 = 14
var Y166 = 14
var Y167 = 14
var Y168 = 14
var Y169 = 14
var Y170 = 14
var Y171 = 14
var Y172 = 14
var Y173 = 14
var Y174 = 14
var Y175 = 14
var Y176 = 14
var Y177 = 14
var Y178 = 14
var Y179 = 14
var Y180 = 14
var Y181 = 14
var Y182 = 14
var Y183 = 14
var Y184 = 14
var Y185 = 14
var Y186 = 14
var Y187 = 14
var Y188 = 14
var Y189 = 14
var Y190 = 14
var Y191 = 14
var Y192 = 14
var Y193 = 14
var Y194 = 14
var Y195 = 14
var Y196 = 14
var Y197 = 14
var Y198 = 14
var Y199 = 14
var Y200 = 14
var Y201 = 14
var Y202 = 14
var Y203 = 14
var Y204 = 14
var Y205 = 14
var Y206 = 14
var Y207 = 14
var Y208 = 14
var Y209 = 14
var Y210 = 14
var Y211 = 14
var Y212 = 14
var Y213 = 14
var Y214 = 14
var Y215 = 14
var Y216 = 14
var Y217 = 14
var Y218 = 14
var Y219 = 14
var Y220 = 14
var Y221 = 14
var Y222 = 14
var Y223 = 14
var Y224 = 14
var Y225 = 14
var Y226 = 14
var Y227 = 14
var Y228 = 14
var Y229 = 14
var Y230 = 14
var Y231 = 14
var Y232 = 14
var Y233 = 14
var Y234 = 14
var Y235 = 14
var Y236 = 14
var Y237 = 14
var Y238 = 14
var Y239 = 14
var Y240 = 14
var Y241 = 14
var Y242 = 14
var Y243 = 14
var Y244 = 14
var Y245 = 14
var Y246 = 14
var Y247 = 14
var Y248 = 14
var Y249 = 14
var Y250 = 14
var Y251 = 14
var Y252 = 14
var Y253 = 14
var Y254 = 14
var Y255 = 14
var Y256 = 14
var Y257 = 14
var Y258 = 14
var Y259 = 14
var Y260 = 14
var Y261 = 14
var Y262 = 14
var Y263 = 14
var Y264 = 14
var Y265 = 14
var Y266 = 14
var Y267 = 14
var Y268 = 14
var Y269 = 14
var Y270 = 14
var Y271 = 14
var Y272 = 14
var Y273 = 14
var Y274 = 14
var Y275 = 14
var Y276 = 14
var Y277 = 14
var Y278 = 14
var Y279 = 14
var Y280 = 14
var Y281 = 14
var Y282 = 14
var Y283 = 14
var Y284 = 14
var Y285 = 14
var Y286 = 14
var Y287 = 14
var Y288 = 14
var Y289 = 14
var Y290 = 14
var Y291 = 14
var Y292 = 14
var Y293 = 14
var Y294 = 14
var Y295 = 14
var Y296 = 14
var Y297 = 14
var Y298 = 14
var Y299 = 14
var Y300 = 14

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
