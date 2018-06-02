package server

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/diff"
	"github.com/haya14busa/reviewdog/doghouse"
)

type Checker struct {
	req *doghouse.CheckRequest
	gh  *github.Client
}

func NewChecker(req *doghouse.CheckRequest, gh *github.Client) *Checker {
	return &Checker{req: req, gh: gh}
}

func (ch *Checker) Check(ctx context.Context) (*doghouse.CheckResponse, error) {
	pr, _, err := ch.gh.PullRequests.Get(ctx, ch.req.Owner, ch.req.Repo, ch.req.PullRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get pr: %v", err)
	}

	headBranch := pr.GetHead().GetRef()
	if headBranch == "" {
		return nil, fmt.Errorf("failed to get branch")
	}

	filediffs, err := ch.diff(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %v", err)
	}

	results := annotationsToCheckResults(ch.req.Annotations)
	filtered := reviewdog.FilterCheck(results, filediffs, 1, "")
	checkRun, err := ch.postCheck(ctx, headBranch, filtered)
	if err != nil {
		return nil, err
	}
	res := &doghouse.CheckResponse{
		ReportURL: checkRun.GetHTMLURL(),
	}
	return res, nil
}

func (ch *Checker) postCheck(ctx context.Context, branch string, checks []*reviewdog.FilteredCheck) (*github.CheckRun, error) {
	var annotations []*github.CheckRunAnnotation
	for _, c := range checks {
		if !c.InDiff {
			continue
		}
		annotations = append(annotations, ch.toCheckRunAnnotation(c))
	}
	conclusion := "success"
	if len(annotations) > 0 {
		conclusion = "action_required"
	}
	name := "reviewdog"
	title := "reviewdog report"
	if ch.req.Name != "" {
		name = ch.req.Name
		title = fmt.Sprintf("reviewdog [%s] report", name)
	}
	opt := github.CreateCheckRunOptions{
		Name:        name,
		ExternalID:  github.String(name),
		HeadBranch:  branch,
		HeadSHA:     ch.req.SHA,
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{time.Now()},
		Output: &github.CheckRunOutput{
			Title:       github.String(title),
			Summary:     github.String(ch.summary(checks)),
			Annotations: annotations,
		},
	}

	checkRun, _, err := ch.gh.Checks.CreateCheckRun(ctx, ch.req.Owner, ch.req.Repo, opt)
	if err != nil {
		return nil, err
	}
	return checkRun, nil
}

func (ch *Checker) summary(checks []*reviewdog.FilteredCheck) string {
	var lines []string
	lines = append(lines, "reported by [reviewdog](https://github.com/haya14busa/reviewdog) :dog:")

	var findings []*reviewdog.FilteredCheck
	var filteredFindings []*reviewdog.FilteredCheck
	for _, c := range checks {
		if c.InDiff {
			findings = append(findings, c)
		} else {
			filteredFindings = append(filteredFindings, c)
		}
	}
	lines = append(lines, ch.summaryFindings("Findings", findings)...)
	lines = append(lines, ch.summaryFindings("Filtered Findings", filteredFindings)...)

	return strings.Join(lines, "\n")
}

func (ch *Checker) summaryFindings(name string, checks []*reviewdog.FilteredCheck) []string {
	var lines []string
	lines = append(lines, "<details>")
	lines = append(lines, fmt.Sprintf("<summary>%s (%d)</summary>", name, len(checks)))
	lines = append(lines, "")
	for _, c := range checks {
		lines = append(lines, ch.buildFindingLink(c))
	}
	lines = append(lines, "</details>")
	return lines
}

func (ch *Checker) buildFindingLink(c *reviewdog.FilteredCheck) string {
	if c.Path == "" {
		return c.Message
	}
	loc := c.Path
	link := fmt.Sprintf("%s", ch.brobHRef(c.Path))
	if c.Lnum != 0 {
		loc = fmt.Sprintf("%s:%d", loc, c.Lnum)
		link = fmt.Sprintf("%s#L%d", link, c.Lnum)
	}
	if c.Col != 0 {
		loc = fmt.Sprintf("%s:%d", loc, c.Col)
	}
	return fmt.Sprintf("[%s](%s): %s", loc, link, c.Message)
}

func (ch *Checker) toCheckRunAnnotation(c *reviewdog.FilteredCheck) *github.CheckRunAnnotation {
	a := &github.CheckRunAnnotation{
		FileName:     github.String(c.Path),
		BlobHRef:     github.String(ch.brobHRef(c.Path)),
		StartLine:    github.Int(c.Lnum),
		EndLine:      github.Int(c.Lnum),
		WarningLevel: github.String("warning"),
		Message:      github.String(c.Message),
	}
	if ch.req.Name != "" {
		a.Title = github.String(fmt.Sprintf("[%s] %s#L%d", ch.req.Name, c.Path, c.Lnum))
	}
	if s := strings.Join(c.Lines, "\n"); s != "" {
		a.RawDetails = github.String(s)
	}
	return a
}

func (ch *Checker) brobHRef(path string) string {
	return fmt.Sprintf("http://github.com/%s/%s/blob/%s/%s", ch.req.Owner, ch.req.Repo, ch.req.SHA, path)
}

func (ch *Checker) diff(ctx context.Context) ([]*diff.FileDiff, error) {
	d, err := ch.rawDiff(ctx)
	if err != nil {
		return nil, err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %v", err)
	}
	return filediffs, nil
}

func (ch *Checker) rawDiff(ctx context.Context) ([]byte, error) {
	opt := github.RawOptions{Type: github.Diff}
	d, _, err := ch.gh.PullRequests.GetRaw(ctx, ch.req.Owner, ch.req.Repo, ch.req.PullRequest, opt)
	if err != nil {
		return nil, err
	}
	return []byte(d), nil
}

func annotationsToCheckResults(as []*doghouse.Annotation) []*reviewdog.CheckResult {
	cs := make([]*reviewdog.CheckResult, 0, len(as))
	for _, a := range as {
		cs = append(cs, &reviewdog.CheckResult{
			Path:    a.Path,
			Lnum:    a.Line,
			Message: a.Message,
			Lines:   strings.Split(a.RawMessage, "\n"),
		})
	}
	return cs
}
