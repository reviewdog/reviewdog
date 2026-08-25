package gitea

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	gitea "gitea.dev/sdk"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/pathutil"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/commentutil"
	"github.com/reviewdog/reviewdog/service/github/githubutils"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ reviewdog.CommentService = (*PullRequest)(nil)
var _ reviewdog.DiffService = (*PullRequest)(nil)

const maxFileComments = 10

const (
	invalidSuggestionPre  = "<details><summary>reviewdog suggestion error</summary>"
	invalidSuggestionPost = "</details>"
)

func isPermissionError(resp *gitea.Response) bool {
	if resp == nil || resp.Response == nil {
		return false
	}
	return resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound
}

// PullRequest is a comment and diff service for Gitea PullRequest.
//
// API:
//
//	https://try.gitea.io/api/swagger#/issue/issueCreateComment
//	POST /repos/:owner/:repo/issues/:number/comments
type PullRequest struct {
	cli      *gitea.Client
	owner    string
	repo     string
	pr       int64
	sha      string
	toolName string

	muComments           sync.Mutex
	maxCommentsPerReview int
	postComments         []*reviewdog.Comment
	logWriter            *githubutils.GitHubActionLogWriter
	fallbackToLog        bool

	postedcs              commentutil.PostedComments
	outdatedComments      map[string]*gitea.PullReviewComment // fingerprint -> comment
	prCommentWithReply    map[int64]bool                      // review id -> bool
	postedIssueComments   map[string]bool                     // fingerprint -> posted
	outdatedIssueComments map[string]*gitea.Comment           // fingerprint -> comment
}

// NewGiteaPullRequest returns a new PullRequest service.
//
// PullRequest service needs git command in $PATH.
//
// The Gitea Token may not have the necessary permissions.
// For example, in the case of a PR from a forked repository.
//
// In such a case, the service will fallback to Gitea Actions workflow commands [1].
//
// [1]: https://docs.gitea.com/usage/actions/comparison
func NewGiteaPullRequest(cli *gitea.Client, owner, repo string, pr int64, sha, level, toolName string) (*PullRequest, error) {
	return &PullRequest{
		cli:       cli,
		owner:     owner,
		repo:      repo,
		pr:        pr,
		sha:       sha,
		toolName:  toolName,
		logWriter: githubutils.NewGitHubActionLogWriter(level),
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// Gitea in parallel.
func (g *PullRequest) Post(_ context.Context, c *reviewdog.Comment) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

func (*PullRequest) ShouldPrependGitRelDir() bool { return true }

// Flush posts comments which has not been posted yet.
func (g *PullRequest) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	defer func() { g.postComments = nil }()

	if err := g.setPostedComment(ctx); err != nil {
		return err
	}
	return g.postAsReviewComment(ctx)
}

// SetTool sets tool name and level to use in comments.
func (g *PullRequest) SetTool(toolName string, level string) {
	g.toolName = toolName
	g.logWriter = githubutils.NewGitHubActionLogWriter(level)
}

// SetMaxCommentsPerReview sets the maximum number of comments to post per review.
func (g *PullRequest) SetMaxCommentsPerReview(max int) {
	g.maxCommentsPerReview = max
}

func (g *PullRequest) postAsReviewComment(ctx context.Context) error {
	if g.fallbackToLog {
		// we don't have permission to post a review comment.
		// Fallback to Gitea Actions log as report.
		for _, c := range g.postComments {
			if err := g.logWriter.Post(ctx, c); err != nil {
				return err
			}
		}
		return g.logWriter.Flush(ctx)
	}

	postComments := g.postComments
	g.postComments = nil
	rawComments := make([]*reviewdog.Comment, 0, len(postComments))
	reviewComments := make([]gitea.CreatePullReviewComment, 0, len(postComments))
	fileComments := make([]gitea.CreateIssueCommentOption, 0)
	remaining := make([]*reviewdog.Comment, 0)
	rootPath, err := serviceutil.GetGitRoot()
	if err != nil {
		return err
	}
	repoBaseHTMLURL, err := g.repoBaseHTMLURL(ctx)
	if err != nil {
		return err
	}
	for _, c := range postComments {
		if !c.Result.InDiffFile {
			// Gitea Review API cannot report results outside diff file. If it's running
			// in Gitea Actions, fallback to Gitea Actions log as report.
			if cienv.IsInGitHubAction() {
				if err := g.logWriter.Post(ctx, c); err != nil {
					return err
				}
			}
			continue
		}
		fprint, err := serviceutil.Fingerprint(c.Result.Diagnostic)
		if err != nil {
			return err
		}
		if g.postedcs.IsPosted(c, giteaCommentLine(c), fprint) || g.postedIssueComments[fprint] {
			// it's already posted. Mark the comment as non-outdated and skip it.
			delete(g.outdatedComments, fprint)
			delete(g.outdatedIssueComments, fprint)
			continue
		}
		rawComments = append(rawComments, c)

		if c.Result.InDiffContext {
			// Only posts maxCommentsPerReview comments per review if option is set.
			if g.maxCommentsPerReview != 0 && len(reviewComments) >= g.maxCommentsPerReview {
				remaining = append(remaining, c)
				continue
			}
			comment := buildReviewComment(c, buildBody(c, repoBaseHTMLURL, rootPath, fprint, g.toolName))
			reviewComments = append(reviewComments, comment)
		} else {
			if len(fileComments) >= maxFileComments {
				remaining = append(remaining, c)
				continue
			}
			comment := gitea.CreateIssueCommentOption{Body: buildBody(c, repoBaseHTMLURL, rootPath, fprint, g.toolName)}
			fileComments = append(fileComments, comment)
		}
	}
	if err := g.logWriter.Flush(ctx); err != nil {
		return err
	}

	if len(reviewComments) > 0 || len(remaining) > 0 {
		// send review comments to Gitea.
		review := gitea.CreatePullReviewOptions{
			CommitID: g.sha,
			State:    gitea.ReviewStateComment,
			Comments: reviewComments,
			Body:     g.remainingCommentsSummary(remaining, repoBaseHTMLURL, rootPath),
		}
		_, resp, err := g.cli.PullRequests.CreatePullReview(ctx, g.owner, g.repo, g.pr, review)
		if err != nil {
			log.Printf("reviewdog: failed to post a review comment: %v", err)
			// Gitea returns 403 or 404 if we don't have permission to post a review comment.
			// fallback to log message in this case.
			if isPermissionError(resp) && cienv.IsInGitHubAction() {
				goto FALLBACK
			}
			return err
		}
	}
	for _, c := range fileComments {
		if _, resp, err := g.cli.Issues.CreateIssueComment(ctx, g.owner, g.repo, g.pr, c); err != nil {
			log.Printf("reviewdog: failed to post a pull request comment: %v", err)
			// Gitea returns 403 or 404 if we don't have permission to post a review comment.
			// fallback to log message in this case.
			if isPermissionError(resp) && cienv.IsInGitHubAction() {
				goto FALLBACK
			}
			return err
		}
	}

	for _, c := range g.outdatedComments {
		if ok := g.prCommentWithReply[c.ID]; ok {
			// Do not remove comment with replies.
			continue
		}
		if _, err := g.cli.Issues.DeleteIssueComment(ctx, g.owner, g.repo, c.ID); err != nil {
			return fmt.Errorf("failed to delete comment (id=%d): %w", c.ID, err)
		}
	}
	for _, c := range g.outdatedIssueComments {
		if _, err := g.cli.Issues.DeleteIssueComment(ctx, g.owner, g.repo, c.ID); err != nil {
			return fmt.Errorf("failed to delete comment (id=%d): %w", c.ID, err)
		}
	}

	return nil

FALLBACK:
	// fallback to Gitea Actions log as report.
	fmt.Fprintln(os.Stderr, `reviewdog: This Gitea Token doesn't have write permission of Review API,
so reviewdog will report results via logging command [1] and create annotations as a fallback.
[1]: https://docs.gitea.com/usage/actions/comparison`)
	g.fallbackToLog = true

	for _, c := range rawComments {
		if err := g.logWriter.Post(ctx, c); err != nil {
			return err
		}
	}
	return g.logWriter.Flush(ctx)
}

func buildReviewComment(c *reviewdog.Comment, body string) gitea.CreatePullReviewComment {
	loc := c.Result.Diagnostic.GetLocation()

	return gitea.CreatePullReviewComment{
		Body:       body,
		Path:       loc.GetPath(),
		NewLineNum: int64(giteaCommentLine(c)),
	}
}

// line represents end line if it's a multiline comment in Gitea, otherwise
// it's start line.
func giteaCommentLine(c *reviewdog.Comment) int {
	if !c.Result.InDiffContext {
		return 0
	}

	_, end := giteaCommentLineRange(c)
	return end
}

func giteaCommentLineRange(c *reviewdog.Comment) (start int, end int) {
	var rdfRange *rdf.Range

	// Prefer first suggestion line range to diagnostic location if available so
	// that reviewdog can post code suggestion as well when the line ranges are
	// different between the diagnostic location and its suggestion.
	if c.Result.FirstSuggestionInDiffContext && len(c.Result.Diagnostic.GetSuggestions()) > 0 {
		rdfRange = c.Result.Diagnostic.GetSuggestions()[0].GetRange()
	} else {
		rdfRange = c.Result.Diagnostic.GetLocation().GetRange()
	}

	startLine := rdfRange.GetStart().GetLine()
	endLine := rdfRange.GetEnd().GetLine()
	if endLine == 0 {
		endLine = startLine
	}
	return int(startLine), int(endLine)
}

func (g *PullRequest) remainingCommentsSummary(remaining []*reviewdog.Comment, baseURL string, gitRootPath string) string {
	if len(remaining) == 0 {
		return ""
	}
	perTool := make(map[string][]*reviewdog.Comment)
	for _, c := range remaining {
		perTool[c.ToolName] = append(perTool[c.ToolName], c)
	}
	var sb strings.Builder
	sb.WriteString("Remaining comments which cannot be posted as a review comment to avoid spamming Pull Request\n")
	sb.WriteString("\n")
	for tool, comments := range perTool {
		sb.WriteString("<details>\n")
		sb.WriteString(fmt.Sprintf("<summary>%s</summary>\n", tool))
		sb.WriteString("\n")
		for _, c := range comments {
			sb.WriteString("<hr>")
			sb.WriteString("\n")
			sb.WriteString("\n")
			sb.WriteString(commentutil.MarkdownComment(c))
			sb.WriteString("\n")
			sb.WriteString("\n")
			sb.WriteString(giteaCodeSnippetURL(baseURL, gitRootPath, c.Result.Diagnostic.GetLocation()))
			sb.WriteString("\n")
			sb.WriteString("\n")
		}
		sb.WriteString("</details>\n")
	}
	return sb.String()
}

// setPostedComment get posted comments from Gitea.
func (g *PullRequest) setPostedComment(ctx context.Context) error {
	g.postedcs = make(commentutil.PostedComments)
	g.outdatedComments = make(map[string]*gitea.PullReviewComment)
	g.prCommentWithReply = make(map[int64]bool)
	g.postedIssueComments = make(map[string]bool)
	g.outdatedIssueComments = make(map[string]*gitea.Comment)
	cs, err := g.comment(ctx)
	if err != nil {
		return err
	}

	commentThreads := make(map[string]int64, len(cs)) // commit/path:line
	for _, c := range cs {
		commentKey := fmt.Sprintf("%s/%s:%d", c.CommitID, c.Path, c.LineNum)
		replyID, ok := commentThreads[commentKey]
		if !ok {
			commentThreads[commentKey] = c.ID
		} else {
			g.prCommentWithReply[replyID] = true
		}

		if meta := serviceutil.ExtractMetaComment(c.Body); meta != nil {
			g.postedcs.AddPostedComment(c.Path, int(c.LineNum), meta.GetFingerprint())
			if meta.SourceName == g.toolName {
				g.outdatedComments[meta.GetFingerprint()] = c // Remove non-outdated comment later.
			}
		}
	}

	issueComments, err := listAllIssueComments(ctx, g.cli, g.owner, g.repo, g.pr,
		gitea.ListIssueCommentOptions{
			ListOptions: gitea.ListOptions{
				Page:     1,
				PageSize: 100,
			},
		})
	if err != nil {
		return err
	}
	for _, c := range issueComments {
		if meta := serviceutil.ExtractMetaComment(c.Body); meta != nil {
			g.postedIssueComments[meta.GetFingerprint()] = true
			if meta.SourceName == g.toolName {
				g.outdatedIssueComments[meta.GetFingerprint()] = c // Remove non-outdated comment later.
			}
		}
	}
	return nil
}

// Diff returns a diff of PullRequest.
func (g *PullRequest) Diff(ctx context.Context) ([]byte, error) {
	return (&PullRequestDiffService{
		Cli:              g.cli,
		Owner:            g.owner,
		Repo:             g.repo,
		PR:               g.pr,
		SHA:              g.sha,
		FallBackToGitCLI: true,
	}).Diff(ctx)
}

// Strip returns 1 as a strip of git diff.
func (g *PullRequest) Strip() int {
	return 1
}

func (g *PullRequest) repoBaseHTMLURL(ctx context.Context) (string, error) {
	repo, _, err := g.cli.Repositories.GetRepo(ctx, g.owner, g.repo)
	if err != nil {
		return "", fmt.Errorf("failed to build repo base HTML URL: %w", err)
	}
	return url.JoinPath(repo.HTMLURL, "src", "commit", g.sha)
}

func (g *PullRequest) comment(ctx context.Context) ([]*gitea.PullReviewComment, error) {
	prs, err := listAllPullRequestReviews(ctx, g.cli, g.owner, g.repo, g.pr, gitea.ListPullReviewsOptions{
		ListOptions: gitea.ListOptions{
			Page:     1,
			PageSize: 100,
		},
	})
	if err != nil {
		return nil, err
	}

	comments := make([]*gitea.PullReviewComment, 0, len(prs))
	for _, pr := range prs {
		c, _, err := g.cli.PullRequests.ListPullReviewComments(ctx, g.owner, g.repo, g.pr, pr.ID)
		if err != nil {
			return nil, err
		}

		comments = append(comments, c...)
	}

	return comments, nil
}

func listAllPullRequestReviews(ctx context.Context, cli *gitea.Client,
	owner, repo string, pr int64, opts gitea.ListPullReviewsOptions,
) ([]*gitea.PullReview, error) {
	reviews, resp, err := cli.PullRequests.ListPullReviews(ctx, owner, repo, pr, opts)
	if err != nil {
		return nil, err
	}

	if resp.NextPage == 0 {
		return reviews, nil
	}

	newOpts := gitea.ListPullReviewsOptions{
		ListOptions: gitea.ListOptions{
			Page:     resp.NextPage,
			PageSize: opts.PageSize,
		},
	}

	restReviews, err := listAllPullRequestReviews(ctx, cli, owner, repo, pr, newOpts)
	if err != nil {
		return nil, err
	}

	return append(reviews, restReviews...), nil
}

func listAllIssueComments(ctx context.Context, cli *gitea.Client,
	owner, repo string, pr int64, opts gitea.ListIssueCommentOptions,
) ([]*gitea.Comment, error) {
	comments, resp, err := cli.Issues.ListIssueComments(ctx, owner, repo, pr, opts)
	if err != nil {
		return nil, err
	}

	if resp.NextPage == 0 {
		return comments, nil
	}

	opts.Page = resp.NextPage
	restComments, err := listAllIssueComments(ctx, cli, owner, repo, pr, opts)
	if err != nil {
		return nil, err
	}

	return append(comments, restComments...), nil
}

func buildBody(c *reviewdog.Comment, baseURL string, gitRootPath string, fprint string, toolName string) string {
	cbody := commentutil.MarkdownComment(c)
	if c.Result.InDiffContext {
		if suggestion := buildSuggestions(c); suggestion != "" {
			cbody += "\n" + suggestion
		}
	} else {
		if c.Result.Diagnostic.GetLocation().GetRange().GetStart().GetLine() > 0 {
			snippetURL := giteaCodeSnippetURL(baseURL, gitRootPath, c.Result.Diagnostic.GetLocation())
			cbody += "\n\n" + snippetURL
		}
	}
	for _, relatedLoc := range c.Result.Diagnostic.GetRelatedLocations() {
		loc := relatedLoc.GetLocation()
		if loc.GetPath() == "" || loc.GetRange().GetStart().GetLine() == 0 {
			continue
		}
		snippetURL := giteaCodeSnippetURL(baseURL, gitRootPath, loc)
		cbody += "\n<hr>\n\n" + relatedLoc.GetMessage() + "\n" + snippetURL
	}
	cbody += fmt.Sprintf("\n%s\n", serviceutil.BuildMetaComment(fprint, toolName))
	return cbody
}

func giteaCodeSnippetURL(baseURL, gitRootPath string, loc *rdf.Location) string {
	relPath := pathutil.NormalizePath(loc.GetPath(), gitRootPath, "")
	relatedURL := fmt.Sprintf("%s/%s", baseURL, relPath)
	if startLine := loc.GetRange().GetStart().GetLine(); startLine > 0 {
		relatedURL += fmt.Sprintf("#L%d", startLine)
	}
	if endLine := loc.GetRange().GetEnd().GetLine(); endLine > 0 {
		relatedURL += fmt.Sprintf("-L%d", endLine)
	}
	return relatedURL
}

func buildSuggestions(c *reviewdog.Comment) string {
	var sb strings.Builder
	for _, s := range c.Result.Diagnostic.GetSuggestions() {
		txt, err := buildSingleSuggestion(c, s)
		if err != nil {
			sb.WriteString(invalidSuggestionPre + err.Error() + invalidSuggestionPost + "\n")
			continue
		}
		sb.WriteString(txt)
		sb.WriteString("\n")
	}
	return sb.String()
}

func buildSingleSuggestion(c *reviewdog.Comment, s *rdf.Suggestion) (string, error) {
	start := s.GetRange().GetStart()
	startLine := int(start.GetLine())
	end := s.GetRange().GetEnd()
	endLine := int(end.GetLine())
	if endLine == 0 {
		endLine = startLine
	}
	gStart, gEnd := giteaCommentLineRange(c)
	if startLine != gStart || endLine != gEnd {
		//lint:ignore ST1005 Gitea is product name
		//nolint:staticcheck
		return "", fmt.Errorf("Gitea comment range and suggestion line range must be same. L%d-L%d v.s. L%d-L%d",
			gStart, gEnd, startLine, endLine)
	}
	if start.GetColumn() > 0 || end.GetColumn() > 0 {
		return buildNonLineBasedSuggestion(c, s)
	}

	txt := s.GetText()
	backticks := commentutil.GetCodeFenceLength(txt)

	var sb strings.Builder
	sb.Grow(backticks + len("suggestion\n") + len(txt) + len("\n") + backticks)
	commentutil.WriteCodeFence(&sb, backticks)
	sb.WriteString("suggestion\n")
	if txt != "" {
		sb.WriteString(txt)
		sb.WriteString("\n")
	}
	commentutil.WriteCodeFence(&sb, backticks)
	return sb.String(), nil
}

func buildNonLineBasedSuggestion(c *reviewdog.Comment, s *rdf.Suggestion) (string, error) {
	sourceLines := c.Result.SourceLines
	if len(sourceLines) == 0 {
		return "", errors.New("source lines are not available")
	}
	start := s.GetRange().GetStart()
	end := s.GetRange().GetEnd()
	startLineContent, err := getSourceLine(sourceLines, int(start.GetLine()))
	if err != nil {
		return "", err
	}
	endLineContent, err := getSourceLine(sourceLines, int(end.GetLine()))
	if err != nil {
		return "", err
	}

	txt := startLineContent[:max(start.GetColumn()-1, 0)] + s.GetText() + endLineContent[max(end.GetColumn()-1, 0):]
	backticks := commentutil.GetCodeFenceLength(txt)

	var sb strings.Builder
	sb.Grow(backticks + len("suggestion\n") + len(txt) + len("\n") + backticks)
	commentutil.WriteCodeFence(&sb, backticks)
	sb.WriteString("suggestion\n")
	sb.WriteString(txt)
	sb.WriteString("\n")
	commentutil.WriteCodeFence(&sb, backticks)
	return sb.String(), nil
}

func getSourceLine(sourceLines map[int]string, line int) (string, error) {
	lineContent, ok := sourceLines[line]
	if !ok {
		return "", fmt.Errorf("source line (L=%d) is not available for this suggestion", line)
	}
	return lineContent, nil
}
