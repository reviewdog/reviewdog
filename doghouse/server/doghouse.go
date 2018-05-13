package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/diff"
	"github.com/haya14busa/reviewdog/doghouse"
)

type DogHouse struct {
	client *http.Client
	req    *doghouse.CheckRequest
	gh     *github.Client
}

func New(req *doghouse.CheckRequest, privateKey []byte, integrationID int, c *http.Client) (*DogHouse, error) {
	dh := &DogHouse{client: c, req: req}
	if err := dh.setGitHubClient(privateKey, integrationID); err != nil {
		return nil, err
	}
	return dh, nil
}

func (dh *DogHouse) Check(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
	pr, _, err := dh.gh.PullRequests.Get(ctx, req.Owner, req.Repo, req.PullRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get pr: %v", err)
	}

	headBranch := pr.GetHead().GetRef()
	if headBranch == "" {
		return nil, fmt.Errorf("failed to get branch")
	}

	filediffs, err := dh.diff(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %v", err)
	}

	results := annotationsToCheckResults(req.Annotations)
	filtered := reviewdog.FilterCheck(results, filediffs, 1, "")
	checkRun, err := dh.postCheck(ctx, headBranch, filtered)
	if err != nil {
		return nil, err
	}
	res := &doghouse.CheckResponse{
		ReportURL: checkRun.GetHTMLURL(),
	}
	return res, nil
}

func (dh *DogHouse) postCheck(ctx context.Context, branch string, checks []*reviewdog.FilteredCheck) (*github.CheckRun, error) {
	var annotations []*github.CheckRunAnnotation
	for _, c := range checks {
		if !c.InDiff {
			continue
		}
		annotations = append(annotations, dh.toCheckRunAnnotation(c))
	}
	conclusion := "success"
	if len(annotations) > 0 {
		conclusion = "action_required"
	}
	name := "reviewdog"
	title := "reviewdog report"
	if dh.req.Name != "" {
		name = dh.req.Name
		title = fmt.Sprintf("reviewdog [%s] report", name)
	}
	opt := github.CreateCheckRunOptions{
		Name:        name,
		ExternalID:  github.String(name),
		HeadBranch:  branch,
		HeadSHA:     dh.req.SHA,
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{time.Now()},
		Output: &github.CheckRunOutput{
			Title:       github.String(title),
			Summary:     github.String(dh.summary(checks)),
			Annotations: annotations,
		},
	}

	checkRun, _, err := dh.gh.Checks.CreateCheckRun(ctx, dh.req.Owner, dh.req.Repo, opt)
	if err != nil {
		return nil, err
	}
	return checkRun, nil
}

func (dh *DogHouse) summary(checks []*reviewdog.FilteredCheck) string {
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
	lines = append(lines, dh.summaryFindings("Findings", findings)...)
	lines = append(lines, dh.summaryFindings("Filtered Findings", filteredFindings)...)

	return strings.Join(lines, "\n")
}

func (dh *DogHouse) summaryFindings(name string, checks []*reviewdog.FilteredCheck) []string {
	var lines []string
	lines = append(lines, "<details>")
	lines = append(lines, fmt.Sprintf("<summary>%s (%d)</summary>", name, len(checks)))
	lines = append(lines, "")
	for _, c := range checks {
		lines = append(lines, dh.buildFindingLink(c))
	}
	lines = append(lines, "</details>")
	return lines
}

func (dh *DogHouse) buildFindingLink(c *reviewdog.FilteredCheck) string {
	if c.Path == "" {
		return c.Message
	}
	loc := c.Path
	link := fmt.Sprintf("%s", dh.brobHRef(c.Path))
	if c.Lnum != 0 {
		loc = fmt.Sprintf("%s:%d", loc, c.Lnum)
		link = fmt.Sprintf("%s#L%d", link, c.Lnum)
	}
	if c.Col != 0 {
		loc = fmt.Sprintf("%s:%d", loc, c.Col)
	}
	return fmt.Sprintf("[%s](%s): %s", loc, link, c.Message)
}

func (dh *DogHouse) toCheckRunAnnotation(c *reviewdog.FilteredCheck) *github.CheckRunAnnotation {
	a := &github.CheckRunAnnotation{
		FileName:     github.String(c.Path),
		BlobHRef:     github.String(dh.brobHRef(c.Path)),
		StartLine:    github.Int(c.Lnum),
		EndLine:      github.Int(c.Lnum),
		WarningLevel: github.String("warning"),
		Message:      github.String(c.Message),
	}
	if dh.req.Name != "" {
		a.Title = github.String(fmt.Sprintf("[%s] %s#L%d", dh.req.Name, c.Path, c.Lnum))
	}
	if s := strings.Join(c.Lines, "\n"); s != "" {
		a.RawDetails = github.String(s)
	}
	return a
}

func (dh *DogHouse) brobHRef(path string) string {
	return fmt.Sprintf("http://github.com/%s/%s/blob/%s/%s", dh.req.Owner, dh.req.Repo, dh.req.SHA, path)
}

func (dh *DogHouse) setGitHubClient(privateKey []byte, integrationID int) error {
	itr, err := ghinstallation.New(dh.client.Transport, integrationID, dh.req.InstallationID, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create gh transport: %v", err)
	}
	dh.client.Transport = itr
	dh.gh = github.NewClient(dh.client)
	return nil
}

func (dh *DogHouse) diff(ctx context.Context) ([]*diff.FileDiff, error) {
	d, err := dh.rawDiff(ctx)
	if err != nil {
		return nil, err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %v", err)
	}
	return filediffs, nil
}

func (dh *DogHouse) rawDiff(ctx context.Context) ([]byte, error) {
	opt := github.RawOptions{Type: github.Diff}
	d, _, err := dh.gh.PullRequests.GetRaw(ctx, dh.req.Owner, dh.req.Repo, dh.req.PullRequest, opt)
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
