package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/github/githubutils"
)

// GitHub check runs API cannot handle too large requests.
// Set max number of filtered findings to be shown in check-run summary.
// ERROR:
//
//	https://api.github.com/repos/easymotion/vim-easymotion/check-runs: 422
//	Invalid request.
//	Only 65535 characters are allowed; 250684 were supplied. []
const maxAllowedSize = 65535

// > The Checks API limits the number of annotations to a maximum of 50 per API
// > request.
// https://developer.github.com/v3/checks/runs/#output-object
const maxAnnotationsPerRequest = 50

var _ reviewdog.CommentService = (*Check)(nil)

type Check struct {
	CLI      *github.Client
	Owner    string
	Repo     string
	PR       int // optional
	SHA      string
	ToolName string
	Level    string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	muResult sync.Mutex
	result   *CheckResult
}

type CheckResult struct {
	ReportURL  string
	Conclusion string
}

func (ch *Check) Post(_ context.Context, c *reviewdog.Comment) error {
	ch.muComments.Lock()
	defer ch.muComments.Unlock()
	ch.postComments = append(ch.postComments, c)
	return nil
}

func (ch *Check) GetResult() *CheckResult {
	ch.muResult.Lock()
	defer ch.muResult.Unlock()
	return ch.result
}

func (ch *Check) Flush(ctx context.Context) error {
	ch.muComments.Lock()
	defer ch.muComments.Unlock()

	check, err := ch.createCheck(ctx)
	if err != nil {
		// If this error is StatusForbidden (403) here, it means reviewdog is
		// running on GitHub Actions and has only read permission (because it's
		// running for Pull Requests from forked repository). If the token itself
		// is invalid, reviewdog should return an error earlier (e.g. when reading
		// Pull Requests diff), so it should be ok not to return error here and
		// return results instead.
		if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusForbidden {
			return errors.New("TODO: graceful degradation here")
		}
		return fmt.Errorf("failed to create check: %w", err)
	}
	checkRun, conclusion, err := ch.postCheck(ctx, check.GetID())
	if err != nil {
		return fmt.Errorf("failed to post result: %w", err)
	}
	ch.muResult.Lock()
	defer ch.muResult.Unlock()
	ch.result = &CheckResult{
		ReportURL:  checkRun.GetHTMLURL(),
		Conclusion: conclusion,
	}
	return nil
}

func (ch *Check) createCheck(ctx context.Context) (*github.CheckRun, error) {
	opt := github.CreateCheckRunOptions{
		Name:    ch.ToolName,
		HeadSHA: ch.SHA,
		Status:  github.String("in_progress"),
	}
	checkRun, _, err := ch.CLI.Checks.CreateCheckRun(ctx, ch.Owner, ch.Repo, opt)
	return checkRun, err
}

func (ch *Check) postCheck(ctx context.Context, checkID int64) (*github.CheckRun, string, error) {
	var annotations []*github.CheckRunAnnotation
	for _, c := range ch.postComments {
		if !c.Result.ShouldReport {
			continue
		}
		annotations = append(annotations, ch.toCheckRunAnnotation(c.Result))
	}
	if len(annotations) > 0 {
		if err := ch.postAnnotations(ctx, checkID, annotations); err != nil {
			return nil, "", fmt.Errorf("failed to post annotations: %w", err)
		}
	}

	conclusion := ch.conclusion(annotations)
	opt := github.UpdateCheckRunOptions{
		Name:        ch.checkName(),
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:   github.String(ch.checkTitle()),
			Summary: github.String(ch.summary(ch.postComments)),
		},
	}
	checkRun, _, err := ch.CLI.Checks.UpdateCheckRun(ctx, ch.Owner, ch.Repo, checkID, opt)
	if err != nil {
		return nil, "", err
	}
	return checkRun, conclusion, nil
}

func (ch *Check) toCheckRunAnnotation(c *filter.FilteredDiagnostic) *github.CheckRunAnnotation {
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

func (ch *Check) buildTitle(c *filter.FilteredDiagnostic) string {
	var sb strings.Builder
	toolName := c.Diagnostic.GetSource().GetName()
	if toolName == "" {
		toolName = ch.ToolName
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

func (ch *Check) postAnnotations(ctx context.Context, checkID int64, annotations []*github.CheckRunAnnotation) error {
	opt := github.UpdateCheckRunOptions{
		Name: ch.checkName(),
		Output: &github.CheckRunOutput{
			Title:       github.String(ch.checkTitle()),
			Summary:     github.String(""), // Post summary with the last request.
			Annotations: annotations[:min(maxAnnotationsPerRequest, len(annotations))],
		},
	}
	if _, _, err := ch.CLI.Checks.UpdateCheckRun(ctx, ch.Owner, ch.Repo, checkID, opt); err != nil {
		return err
	}
	if len(annotations) > maxAnnotationsPerRequest {
		return ch.postAnnotations(ctx, checkID, annotations[maxAnnotationsPerRequest:])
	}
	return nil
}

// https://developer.github.com/v3/checks/runs/#parameters-1
func (ch *Check) conclusion(annotations []*github.CheckRunAnnotation) string {
	checkResult := "success"

	if ch.Level != "" {
		// Level takes precedence when configured (for backwards compatibility)
		if len(annotations) == 0 {
			return checkResult
		}
		switch strings.ToLower(ch.Level) {
		case "info", "warning":
			return "neutral"
		}
		return "failure"
	} else {
		precedence := map[string]int{
			"success": 0,
			"notice":  1,
			"warning": 2,
			"failure": 3,
		}

		highestLevel := "success"
		for _, a := range annotations {
			annotationLevel := *a.AnnotationLevel
			if precedence[annotationLevel] > precedence[highestLevel] {
				highestLevel = annotationLevel
			}
		}
		checkResult = highestLevel
	}

	switch checkResult {
	case "success":
		return "success"
	case "notice", "warning":
		return "neutral"
	}
	return "failure"
}

func (ch *Check) checkName() string {
	if ch.ToolName != "" {
		return ch.ToolName
	}
	return "reviewdog"
}

func (ch *Check) checkTitle() string {
	if name := ch.checkName(); name != "reviewdog" {
		return fmt.Sprintf("reviewdog [%s] report", name)
	}
	return "reviewdog report"
}

// https://developer.github.com/v3/checks/runs/#annotations-object
func (ch *Check) annotationLevel(s rdf.Severity) string {
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

func (ch *Check) reqAnnotationLevel() string {
	switch strings.ToLower(ch.Level) {
	case "info":
		return "notice"
	case "warning":
		return "warning"
	case "failure":
		return "failure"
	}
	return "failure"
}

func (ch *Check) summary(checks []*reviewdog.Comment) string {
	var lines []string
	var usedBytes int
	lines = append(lines, "reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:")
	usedBytes += len(lines[0]) + 1
	var findings []*filter.FilteredDiagnostic
	var filteredFindings []*filter.FilteredDiagnostic
	for _, c := range checks {
		if c.Result.ShouldReport {
			findings = append(findings, c.Result)
		} else {
			filteredFindings = append(filteredFindings, c.Result)
		}
	}

	findingMsgs, usedBytes := ch.summaryFindings("Findings", usedBytes, findings)
	lines = append(lines, findingMsgs...)
	filteredFindingsMsgs, _ := ch.summaryFindings("Filtered Findings", usedBytes, filteredFindings)
	lines = append(lines, filteredFindingsMsgs...)
	return strings.Join(lines, "\n")
}

func (ch *Check) summaryFindings(name string, usedBytes int, checks []*filter.FilteredDiagnostic) ([]string, int) {
	var lines []string
	lines = append(lines, fmt.Sprintf("<details>\n<summary>%s (%d)</summary>\n", name, len(checks)))
	if len(lines[0])+1+usedBytes > maxAllowedSize {
		// bail out if we're already over the limit
		return nil, usedBytes
	}
	usedBytes += len(lines[0]) + 1
	for _, c := range checks {
		nextLine := githubutils.LinkedMarkdownDiagnostic(ch.Owner, ch.Repo, ch.SHA, c.Diagnostic)
		// existing lines + newline + closing details tag must be smaller than the max allowed size
		if usedBytes+len(nextLine)+1+10 >= maxAllowedSize {
			cutoffMsg := "... (Too many findings. Dropped some findings)"
			if usedBytes+len(cutoffMsg)+1+10 <= maxAllowedSize {
				lines = append(lines, cutoffMsg)
				usedBytes += len(cutoffMsg) + 1
			}
			break
		}
		lines = append(lines, nextLine)
		usedBytes += len(nextLine) + 1
	}
	lines = append(lines, "</details>")
	usedBytes += 10 + 1
	return lines, usedBytes
}
