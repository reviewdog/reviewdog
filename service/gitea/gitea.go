package gitea

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"code.gitea.io/sdk/gitea"
	"google.golang.org/protobuf/proto"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/pathutil"
	"github.com/reviewdog/reviewdog/proto/metacomment"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/commentutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ reviewdog.CommentService = (*PullRequest)(nil)
var _ reviewdog.DiffService = (*PullRequest)(nil)

const (
	invalidSuggestionPre  = "<details><summary>reviewdog suggestion error</summary>"
	invalidSuggestionPost = "</details>"
)

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

	postedcs           commentutil.PostedComments
	outdatedComments   map[string]*gitea.PullReviewComment // fingerprint -> comment
	prCommentWithReply map[int64]bool                      // review id -> bool

	// wd is working directory relative to root of repository.
	wd string
}

// NewGiteaPullRequest returns a new PullRequest service.
//
// PullRequest service needs git command in $PATH.
func NewGiteaPullRequest(cli *gitea.Client, owner, repo string, pr int64, sha, toolName string) (*PullRequest, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("pull request needs 'git' command: %w", err)
	}
	return &PullRequest{
		cli:      cli,
		owner:    owner,
		repo:     repo,
		pr:       pr,
		sha:      sha,
		toolName: toolName,
		wd:       workDir,
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// Gitea in parallel.
func (g *PullRequest) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(filepath.Join(g.wd,
		c.Result.Diagnostic.GetLocation().GetPath()))
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (g *PullRequest) Flush(_ context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	defer func() { g.postComments = nil }()

	if err := g.setPostedComment(); err != nil {
		return err
	}
	return g.postAsReviewComment()
}

// SetTool sets tool name to use in comments.
func (g *PullRequest) SetTool(toolName string, _ string) {
	g.toolName = toolName
}

// SetMaxCommentsPerReview sets the maximum number of comments to post per review.
func (g *PullRequest) SetMaxCommentsPerReview(max int) {
	g.maxCommentsPerReview = max
}

func (g *PullRequest) postAsReviewComment() error {
	postComments := g.postComments
	g.postComments = nil
	reviewComments := make([]gitea.CreatePullReviewComment, 0, len(postComments))
	remaining := make([]*reviewdog.Comment, 0)
	rootPath, err := serviceutil.GetGitRoot()
	if err != nil {
		return err
	}
	repoBaseHTMLURL, err := g.repoBaseHTMLURL()
	if err != nil {
		return err
	}
	for _, c := range postComments {
		if !c.Result.InDiffFile {
			continue
		}
		fprint, err := fingerprint(c.Result.Diagnostic)
		if err != nil {
			return err
		}
		if g.postedcs.IsPosted(c, giteaCommentLine(c), fprint) {
			// it's already posted. Mark the comment as non-outdated and skip it.
			delete(g.outdatedComments, fprint)
			continue
		}

		if !c.Result.InDiffContext {
			// If the result is outside of diff context, skip it.
			continue
		}

		// Only posts maxCommentsPerReview comments per review if option is set.
		if g.maxCommentsPerReview != 0 && len(reviewComments) >= g.maxCommentsPerReview {
			remaining = append(remaining, c)
			continue
		}
		comment := buildReviewComment(c, buildBody(c, repoBaseHTMLURL, rootPath, fprint, g.toolName))
		reviewComments = append(reviewComments, comment)
	}

	if len(reviewComments) > 0 || len(remaining) > 0 {
		// send review comments to Gitea.
		review := gitea.CreatePullReviewOptions{
			CommitID: g.sha,
			State:    gitea.ReviewStateComment,
			Comments: reviewComments,
			Body:     g.remainingCommentsSummary(remaining, repoBaseHTMLURL, rootPath),
		}
		_, _, err := g.cli.CreatePullReview(g.owner, g.repo, g.pr, review)
		if err != nil {
			log.Printf("reviewdog: failed to post a review comment: %v", err)
			return err
		}
	}

	for _, c := range g.outdatedComments {
		if ok := g.prCommentWithReply[c.ID]; ok {
			// Do not remove comment with replies.
			continue
		}
		if _, err := g.cli.DeleteIssueComment(g.owner, g.repo, c.ID); err != nil {
			return fmt.Errorf("failed to delete comment (id=%d): %w", c.ID, err)
		}
	}

	return nil
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
func (g *PullRequest) setPostedComment() error {
	g.postedcs = make(commentutil.PostedComments)
	g.outdatedComments = make(map[string]*gitea.PullReviewComment)
	g.prCommentWithReply = make(map[int64]bool)
	cs, err := g.comment()
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

		if meta := extractMetaComment(c.Body); meta != nil {
			g.postedcs.AddPostedComment(c.Path, int(c.LineNum), meta.GetFingerprint())
			if meta.SourceName == g.toolName {
				g.outdatedComments[meta.GetFingerprint()] = c // Remove non-outdated comment later.
			}
		}
	}
	return nil
}

func extractMetaComment(body string) *metacomment.MetaComment {
	prefix := "<!-- __reviewdog__:"
	for _, line := range strings.Split(body, "\n") {
		if after, found := strings.CutPrefix(line, prefix); found {
			if metastring, foundSuffix := strings.CutSuffix(after, " -->"); foundSuffix {
				meta, err := DecodeMetaComment(metastring)
				if err != nil {
					log.Printf("failed to decode MetaComment: %v", err)
					continue
				}
				return meta
			}
		}
	}
	return nil
}

// DecodeMetaComment decodes a base64 encoded meta comment.
func DecodeMetaComment(metaBase64 string) (*metacomment.MetaComment, error) {
	b, err := base64.StdEncoding.DecodeString(metaBase64)
	if err != nil {
		return nil, err
	}
	meta := &metacomment.MetaComment{}
	if err := proto.Unmarshal(b, meta); err != nil {
		return nil, err
	}
	return meta, nil
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

func (g *PullRequest) repoBaseHTMLURL() (string, error) {
	repo, _, err := g.cli.GetRepo(g.owner, g.repo)
	if err != nil {
		return "", fmt.Errorf("failed to build repo base HTML URL: %w", err)
	}
	return url.JoinPath(repo.HTMLURL, "src", "commit", g.sha)
}

func (g *PullRequest) comment() ([]*gitea.PullReviewComment, error) {
	prs, err := listAllPullRequestReviews(g.cli, g.owner, g.repo, g.pr, gitea.ListPullReviewsOptions{
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
		c, _, err := g.cli.ListPullReviewComments(g.owner, g.repo, g.pr, pr.ID)
		if err != nil {
			return nil, err
		}

		comments = append(comments, c...)
	}

	return comments, nil
}

func listAllPullRequestReviews(cli *gitea.Client,
	owner, repo string, pr int64, opts gitea.ListPullReviewsOptions,
) ([]*gitea.PullReview, error) {
	reviews, resp, err := cli.ListPullReviews(owner, repo, pr, opts)
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

	restReviews, err := listAllPullRequestReviews(cli, owner, repo, pr, newOpts)
	if err != nil {
		return nil, err
	}

	return append(reviews, restReviews...), nil
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
	cbody += fmt.Sprintf("\n<!-- __reviewdog__:%s -->\n", BuildMetaComment(fprint, toolName))
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

// BuildMetaComment builds a base64 encoded meta comment with the given fingerprint and tool name.
func BuildMetaComment(fprint string, toolName string) string {
	b, _ := proto.Marshal(
		&metacomment.MetaComment{
			Fingerprint: fprint,
			SourceName:  toolName,
		},
	)
	return base64.StdEncoding.EncodeToString(b)
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

func fingerprint(d *rdf.Diagnostic) (string, error) {
	h := fnv.New64a()
	// Ideally, we should not use proto.Marshal since Proto Serialization Is Not
	// Canonical.
	// https://protobuf.dev/programming-guides/serialization-not-canonical/
	//
	// However, I left it as-is for now considering the same reviewdog binary
	// should re-calculate and compare fingerprint for almost all cases.
	data, err := proto.Marshal(d)
	if err != nil {
		return "", err
	}
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum64()), nil
}
