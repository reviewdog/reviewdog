package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/doghouse"
	"golang.org/x/sync/errgroup"
)

// GitHub check runs API cannot handle too large requests.
// Set max number of filtered findings to be showen in check-run summary.
// ERROR:
//  https://api.github.com/repos/easymotion/vim-easymotion/check-runs: 422
//  Invalid request.
//  Only 65535 characters are allowed; 250684 were supplied. []
const maxFilteredFinding = 150

type Checker struct {
	req *doghouse.CheckRequest
	gh  checkerGitHubClientInterface
}

func NewChecker(req *doghouse.CheckRequest, gh *github.Client) *Checker {
	return &Checker{req: req, gh: &checkerGitHubClient{Client: gh}}
}

func (ch *Checker) Check(ctx context.Context) (*doghouse.CheckResponse, error) {
	var branch string
	var filediffs []*diff.FileDiff

	// Get branch from PullRequest API and PullRequest diff from diff API
	// concurrently.
	eg, ctx4eg := errgroup.WithContext(ctx)
	eg.Go(func() error {
		br, err := ch.getBranch(ctx4eg)
		if err != nil {
			return err
		}
		if br == "" {
			return fmt.Errorf("failed to get branch")
		}
		branch = br
		return nil
	})
	eg.Go(func() error {
		fd, err := ch.diff(ctx4eg)
		if err != nil {
			return fmt.Errorf("fail to parse diff: %v", err)
		}
		filediffs = fd
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to get branch/diff: %v", err)
	}

	results := annotationsToCheckResults(ch.req.Annotations)
	filtered := reviewdog.FilterCheck(results, filediffs, 1, "")
	checkRun, err := ch.postCheck(ctx, branch, filtered)
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
		return nil, fmt.Errorf("failed to post result: %v", err)
	}
	res := &doghouse.CheckResponse{
		ReportURL: checkRun.GetHTMLURL(),
	}
	return res, nil
}

func (ch *Checker) getBranch(ctx context.Context) (string, error) {
	if ch.req.Branch != "" {
		return ch.req.Branch, nil
	}
	pr, err := ch.gh.GetPullRequest(ctx, ch.req.Owner, ch.req.Repo, ch.req.PullRequest)
	if err != nil {
		return "", fmt.Errorf("failed to get pr: %v", err)
	}
	return pr.GetHead().GetRef(), nil
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
		conclusion = ch.conclusion()
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
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:       github.String(title),
			Summary:     github.String(ch.summary(checks)),
			Annotations: annotations,
		},
	}
	return ch.gh.CreateCheckRun(ctx, ch.req.Owner, ch.req.Repo, opt)
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
func (ch *Checker) annotationLevel() string {
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

func (ch *Checker) summary(checks []*reviewdog.FilteredCheck) string {
	var lines []string
	lines = append(lines, "reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:")

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
	for i, c := range checks {
		if i >= maxFilteredFinding {
			lines = append(lines, "... (Too many findings. Dropped some findings)")
			break
		}
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
	link := ch.brobHRef(c.Path)
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
		Path:            github.String(c.Path),
		BlobHRef:        github.String(ch.brobHRef(c.Path)),
		StartLine:       github.Int(c.Lnum),
		EndLine:         github.Int(c.Lnum),
		AnnotationLevel: github.String(ch.annotationLevel()),
		Message:         github.String(c.Message),
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
	d, err := ch.gh.GetPullRequestDiff(ctx, ch.req.Owner, ch.req.Repo, ch.req.PullRequest)
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
