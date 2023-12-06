package gitea

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/commentutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"

	"code.gitea.io/sdk/gitea"
)

var (
	_ reviewdog.CommentService = &PullRequest{}
	_ reviewdog.DiffService    = &PullRequest{}
)

// PullRequest is a comment and diff service for Gitea PullRequest.
//
// API:
//
//	https://try.gitea.io/api/swagger#/issue/issueCreateComment
//	POST /repos/:owner/:repo/issues/:number/comments
type PullRequest struct {
	cli   *gitea.Client
	owner string
	repo  string
	pr    int64
	sha   string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	postedcs commentutil.PostedComments

	// wd is working directory relative to root of repository.
	wd string
}

// NewGiteaPullRequest returns a new PullRequest service.
// PullRequest service needs git command in $PATH.
func NewGiteaPullRequest(cli *gitea.Client, owner, repo string, pr int64, sha string) (*PullRequest, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("pull request needs 'git' command: %w", err)
	}
	return &PullRequest{
		cli:   cli,
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
		wd:    workDir,
	}, nil
}

// Diff returns a diff of PullRequest.
func (g *PullRequest) Diff(_ context.Context) ([]byte, error) {
	diff, _, err := g.cli.GetPullRequestDiff(g.owner, g.repo, g.pr, gitea.PullRequestDiffOptions{
		Binary: false,
	})
	if err != nil {
		return nil, err
	}

	return diff, nil
}

// Strip returns 1 as a strip of git diff.
func (g *PullRequest) Strip() int {
	return 1
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

	if err := g.setPostedComment(); err != nil {
		return err
	}
	return g.postAsReviewComment()
}

// setPostedComment get posted comments from Gitea.
func (g *PullRequest) setPostedComment() error {
	g.postedcs = make(commentutil.PostedComments)

	cs, err := g.comment()
	if err != nil {
		return err
	}

	for _, c := range cs {
		if c.LineNum == 0 || c.Path == "" || c.Body == "" {
			continue
		}
		g.postedcs.AddPostedComment(c.Path, int(c.LineNum), c.Body)
	}

	return nil
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
	prs, resp, err := cli.ListPullReviews(owner, repo, pr, opts)
	if err != nil {
		return nil, err
	}

	if resp.NextPage == 0 {
		return prs, nil
	}

	newOpts := gitea.ListPullReviewsOptions{
		ListOptions: gitea.ListOptions{
			Page:     resp.NextPage,
			PageSize: opts.PageSize,
		},
	}

	restPrs, err := listAllPullRequestReviews(cli, owner, repo, pr, newOpts)
	if err != nil {
		return nil, err
	}

	return append(prs, restPrs...), nil
}

func (g *PullRequest) postAsReviewComment() error {
	postComments := g.postComments
	g.postComments = nil
	reviewComments := make([]gitea.CreatePullReviewComment, 0, len(postComments))

	for _, comment := range postComments {
		if !comment.Result.InDiffFile {
			continue
		}

		body := commentutil.MarkdownComment(comment)
		if g.postedcs.IsPosted(comment, giteaCommentLine(comment), body) {
			// it's already posted. skip it.
			continue
		}

		if !comment.Result.InDiffContext {
			// If the result is outside of diff context, skip it.
			continue
		}

		reviewComments = append(reviewComments, buildReviewComment(comment, body))
	}

	if len(reviewComments) > 0 {
		// send review comments to Gitea.
		review := gitea.CreatePullReviewOptions{
			CommitID: g.sha,
			State:    gitea.ReviewStateComment,
			Comments: reviewComments,
		}
		_, _, err := g.cli.CreatePullReview(g.owner, g.repo, g.pr, review)
		if err != nil {
			return err
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

	loc := c.Result.Diagnostic.GetLocation()
	startLine := loc.GetRange().GetStart().GetLine()
	endLine := loc.GetRange().GetEnd().GetLine()

	if endLine == 0 {
		endLine = startLine
	}

	return int(endLine)
}
