package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v38/github"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/github/githubutils"
)

// GitHub check runs API cannot handle too large requests.
// Set max number of filtered findings to be shown in check-run summary.
// ERROR:
//  https://api.github.com/repos/easymotion/vim-easymotion/check-runs: 422
//  Invalid request.
//  Only 65535 characters are allowed; 250684 were supplied. []
const maxFilteredFinding = 150

// > The Checks API limits the number of annotations to a maximum of 50 per API
// > request.
// https://developer.github.com/v3/checks/runs/#output-object
const maxAnnotationsPerRequest = 50

type Checker struct {
	req *doghouse.CheckRequest
	gh  checkerGitHubClientInterface
}

func NewChecker(req *doghouse.CheckRequest, gh *github.Client) *Checker {
	return &Checker{req: req, gh: &checkerGitHubClient{Client: gh}}
}

func (ch *Checker) Check(ctx context.Context) (*doghouse.CheckResponse, error) {
	var filediffs []*diff.FileDiff
	if ch.req.PullRequest != 0 {
		var err error
		filediffs, err = ch.pullRequestDiff(ctx, ch.req.PullRequest)
		if err != nil {
			return nil, fmt.Errorf("fail to parse diff: %w", err)
		}
	}

	results := annotationsToDiagnostics(ch.req.Annotations)
	filterMode := ch.req.FilterMode
	//lint:ignore SA1019 Need to support OutsideDiff for backward compatibility.
	if ch.req.PullRequest == 0 || ch.req.OutsideDiff {
		// If it's not Pull Request run, do not filter results by diff regardless
		// of the filter mode.
		filterMode = filter.ModeNoFilter
	}
	filtered := filter.FilterCheck(results, filediffs, 1, "", filterMode)
	check, err := ch.createCheck(ctx)
	if err != nil {
		// If this error is StatusForbidden (403) here, it means reviewdog is
		// running on GitHub Actions and has only read permission (because it's
		// running for Pull Requests from forked repository). If the token itself
		// is invalid, reviewdog should return an error earlier (e.g. when reading
		// Pull Requests diff), so it should be ok not to return error here and
		// return results instead.
		if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusForbidden {
			return &doghouse.CheckResponse{CheckedResults: filtered}, nil
		}
		return nil, fmt.Errorf("failed to create check: %w", err)
	}

	checkRun, conclusion, err := ch.postCheck(ctx, check.GetID(), filtered)
	if err != nil {
		return nil, fmt.Errorf("failed to post result: %w", err)
	}
	res := &doghouse.CheckResponse{
		ReportURL:  checkRun.GetHTMLURL(),
		Conclusion: conclusion,
	}
	return res, nil
}

func (ch *Checker) postCheck(ctx context.Context, checkID int64, checks []*filter.FilteredDiagnostic) (*github.CheckRun, string, error) {
	var annotations []*github.CheckRunAnnotation
	for _, c := range checks {
		if !c.ShouldReport {
			continue
		}
		annotations = append(annotations, ch.toCheckRunAnnotation(c))
	}
	if len(annotations) > 0 {
		if err := ch.postAnnotations(ctx, checkID, annotations); err != nil {
			return nil, "", fmt.Errorf("failed to post annotations: %w", err)
		}
	}

	conclusion := "success"
	if len(annotations) > 0 {
		conclusion = ch.conclusion()
	}
	opt := github.UpdateCheckRunOptions{
		Name:        ch.checkName(),
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:   github.String(ch.checkTitle()),
			Summary: github.String(ch.summary(checks)),
		},
	}
	checkRun, err := ch.gh.UpdateCheckRun(ctx, ch.req.Owner, ch.req.Repo, checkID, opt)
	if err != nil {
		return nil, "", err
	}
	return checkRun, conclusion, nil
}

func (ch *Checker) createCheck(ctx context.Context) (*github.CheckRun, error) {
	opt := github.CreateCheckRunOptions{
		Name:    ch.checkName(),
		HeadSHA: ch.req.SHA,
		Status:  github.String("in_progress"),
	}
	return ch.gh.CreateCheckRun(ctx, ch.req.Owner, ch.req.Repo, opt)
}

func (ch *Checker) postAnnotations(ctx context.Context, checkID int64, annotations []*github.CheckRunAnnotation) error {
	opt := github.UpdateCheckRunOptions{
		Name: ch.checkName(),
		Output: &github.CheckRunOutput{
			Title:       github.String(ch.checkTitle()),
			Summary:     github.String(""), // Post summary with the last request.
			Annotations: annotations[:min(maxAnnotationsPerRequest, len(annotations))],
		},
	}
	if _, err := ch.gh.UpdateCheckRun(ctx, ch.req.Owner, ch.req.Repo, checkID, opt); err != nil {
		return err
	}
	if len(annotations) > maxAnnotationsPerRequest {
		return ch.postAnnotations(ctx, checkID, annotations[maxAnnotationsPerRequest:])
	}
	return nil
}

func (ch *Checker) checkName() string {
	if ch.req.Name != "" {
		return ch.req.Name
	}
	return "reviewdog"
}

func (ch *Checker) checkTitle() string {
	if name := ch.checkName(); name != "reviewdog" {
		return fmt.Sprintf("reviewdog [%s] report", name)
	}
	return "reviewdog report"
}

// https://developer.github.com/v3/checks/runs/#parameters-1
func (ch *Checker) conclusion() string {
	switch strings.ToLower(ch.req.Level) {
	case "info", "warning":
		return "neutral"
	}
	return "failure"
}

// https://developer.github.com/v3/checks/runs/#annotations-object
func (ch *Checker) annotationLevel(s rdf.Severity) string {
	switch s {
	case rdf.Severity_ERROR:
		return "failure"
	case rdf.Severity_WARNING:
		return "warning"
	case rdf.Severity_INFO:
		return "notice"
	default:
		return ch.reqAnnotationLevel()
	}
}

func (ch *Checker) reqAnnotationLevel() string {
	switch strings.ToLower(ch.req.Level) {
	case "info":
		return "notice"
	case "warning":
		return "warning"
	case "failure":
		return "failure"
	}
	return "failure"
}

func (ch *Checker) summary(checks []*filter.FilteredDiagnostic) string {
	var lines []string
	lines = append(lines, "reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:")

	var findings []*filter.FilteredDiagnostic
	var filteredFindings []*filter.FilteredDiagnostic
	for _, c := range checks {
		if c.ShouldReport {
			findings = append(findings, c)
		} else {
			filteredFindings = append(filteredFindings, c)
		}
	}
	lines = append(lines, ch.summaryFindings("Findings", findings)...)
	lines = append(lines, ch.summaryFindings("Filtered Findings", filteredFindings)...)

	return strings.Join(lines, "\n")
}

func (ch *Checker) summaryFindings(name string, checks []*filter.FilteredDiagnostic) []string {
	var lines []string
	lines = append(lines, "<details>")
	lines = append(lines, fmt.Sprintf("<summary>%s (%d)</summary>", name, len(checks)))
	lines = append(lines, "")
	for i, c := range checks {
		if i >= maxFilteredFinding {
			lines = append(lines, "... (Too many findings. Dropped some findings)")
			break
		}
		lines = append(lines, githubutils.LinkedMarkdownDiagnostic(
			ch.req.Owner, ch.req.Repo, ch.req.SHA, c.Diagnostic))
	}
	lines = append(lines, "</details>")
	return lines
}

func (ch *Checker) toCheckRunAnnotation(c *filter.FilteredDiagnostic) *github.CheckRunAnnotation {
	loc := c.Diagnostic.GetLocation()
	startLine := int(loc.GetRange().GetStart().GetLine())
	endLine := int(loc.GetRange().GetEnd().GetLine())
	if endLine == 0 {
		endLine = startLine
	}
	a := &github.CheckRunAnnotation{
		Path:            github.String(loc.GetPath()),
		StartLine:       github.Int(startLine),
		EndLine:         github.Int(endLine),
		AnnotationLevel: github.String(ch.annotationLevel(c.Diagnostic.Severity)),
		Message:         github.String(c.Diagnostic.GetMessage()),
		Title:           github.String(ch.buildTitle(c)),
	}
	// Annotations only support start_column and end_column on the same line.
	if startLine == endLine {
		if s, e := loc.GetRange().GetStart().GetColumn(), loc.GetRange().GetEnd().GetColumn(); s != 0 && e != 0 {
			a.StartColumn = github.Int(int(s))
			a.EndColumn = github.Int(int(e))
		}
	}
	if s := c.Diagnostic.GetOriginalOutput(); s != "" {
		a.RawDetails = github.String(s)
	}
	return a
}

func (ch *Checker) buildTitle(c *filter.FilteredDiagnostic) string {
	var sb strings.Builder
	toolName := c.Diagnostic.GetSource().GetName()
	if toolName == "" {
		toolName = ch.req.Name
	}
	if toolName != "" {
		sb.WriteString(fmt.Sprintf("[%s] ", toolName))
	}
	loc := c.Diagnostic.GetLocation()
	sb.WriteString(loc.GetPath())
	if startLine := int(loc.GetRange().GetStart().GetLine()); startLine > 0 {
		sb.WriteString(fmt.Sprintf("#L%d", startLine))
		if endLine := int(loc.GetRange().GetEnd().GetLine()); startLine < endLine {
			sb.WriteString(fmt.Sprintf("-L%d", endLine))
		}
	}
	if code := c.Diagnostic.GetCode().GetValue(); code != "" {
		if url := c.Diagnostic.GetCode().GetUrl(); url != "" {
			sb.WriteString(fmt.Sprintf(" <%s>(%s)", code, url))
		} else {
			sb.WriteString(fmt.Sprintf(" <%s>", code))
		}
	}
	return sb.String()
}

func (ch *Checker) pullRequestDiff(ctx context.Context, pr int) ([]*diff.FileDiff, error) {
	d, err := ch.rawPullRequestDiff(ctx, pr)
	if err != nil {
		return nil, err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %w", err)
	}
	return filediffs, nil
}

func (ch *Checker) rawPullRequestDiff(ctx context.Context, pr int) ([]byte, error) {
	d, err := ch.gh.GetPullRequestDiff(ctx, ch.req.Owner, ch.req.Repo, pr)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func annotationsToDiagnostics(as []*doghouse.Annotation) []*rdf.Diagnostic {
	ds := make([]*rdf.Diagnostic, 0, len(as))
	for _, a := range as {
		ds = append(ds, annotationToDiagnostic(a))
	}
	return ds
}

func annotationToDiagnostic(a *doghouse.Annotation) *rdf.Diagnostic {
	if a.Diagnostic != nil {
		return a.Diagnostic
	}
	// Old reviwedog CLI doesn't have the Diagnostic field.
	return &rdf.Diagnostic{
		Location: &rdf.Location{
			Path: a.Path,
			Range: &rdf.Range{
				Start: &rdf.Position{
					Line: int32(a.Line),
				},
			},
		},
		Message:        a.Message,
		OriginalOutput: a.RawMessage,
	}
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
