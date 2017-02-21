package reviewdog

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-github/github"
	"context"
)

var _ = github.ScopeAdminOrg

var _ CommentService = &GitHubPullRequest{}
var _ DiffService = &GitHubPullRequest{}

// `path` to `position`(Lnum for new file) to comment `body`s
type postedcomments map[string]map[int][]string

// IsPosted returns true if a given comment has been posted in GitHub already,
// otherwise returns false. It sees comments with same path, same position,
// and same body as same comments.
func (p postedcomments) IsPosted(c *Comment) bool {
	if _, ok := p[c.Path]; !ok {
		return false
	}
	bodys, ok := p[c.Path][c.LnumDiff]
	if !ok {
		return false
	}
	for _, body := range bodys {
		if body == commentBody(c) {
			return true
		}
	}
	return false
}

// GitHubPullRequest is a comment and diff service for GitHub PullRequest.
//
// API:
//	https://developer.github.com/v3/pulls/comments/#create-a-comment
// 	POST /repos/:owner/:repo/pulls/:number/comments
type GitHubPullRequest struct {
	postComments []*Comment

	cli   *github.Client
	ctx   context.Context
	owner string
	repo  string
	pr    int
	sha   string

	postedcs postedcomments

	muFlash sync.Mutex
}

// NewGitHubPullReqest returns a new GitHubPullRequest service.
func NewGitHubPullReqest(cli *github.Client, ctx context.Context, owner, repo string, pr int, sha string) *GitHubPullRequest {
	return &GitHubPullRequest{
		cli:   cli,
		ctx:   ctx,
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
}

// Post accepts a comment and holds it. Flash method actually posts comments to
// GitHub in parallel.
func (g *GitHubPullRequest) Post(c *Comment) error {
	g.muFlash.Lock()
	defer g.muFlash.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

const bodyPrefix = `<sub>reported by [reviewdog](https://github.com/haya14busa/reviewdog) :dog:</sub>`

func commentBody(c *Comment) string {
	tool := ""
	if c.ToolName != "" {
		tool = fmt.Sprintf("**[%s]** ", c.ToolName)
	}
	return tool + bodyPrefix + "\n" + c.Body
}

var githubAPIHost = "api.github.com"

// Flash posts comments which has not been posted yet.
func (g *GitHubPullRequest) Flash() error {
	g.muFlash.Lock()
	defer g.muFlash.Unlock()

	if err := g.setPostedComment(); err != nil {
		return err
	}
	// TODO(haya14busa,#58): remove host check when GitHub Enterprise supports
	// Pull Request API.
	if g.cli.BaseURL.Host == githubAPIHost {
		return g.postAsReviewComment()
	}
	return g.postCommentsForEach()
}

func (g *GitHubPullRequest) postAsReviewComment() error {
	comments := make([]*github.DraftReviewComment, 0, len(g.postComments))
	for _, c := range g.postComments {
		if g.postedcs.IsPosted(c) {
			continue
		}
		cbody := commentBody(c)
		comments = append(comments, &github.DraftReviewComment{
			Path:     &c.Path,
			Position: &c.LnumDiff,
			Body:     &cbody,
		})
	}

	if len(comments) == 0 {
		return nil
	}

	// TODO(haya14busa): it might be useful to report overview results by "body"
	// field.
	review := &github.PullRequestReviewRequest{
		Event:    github.String("COMMENT"),
		Comments: comments,
	}
	_, _, err := g.cli.PullRequests.CreateReview(g.ctx, g.owner, g.repo, g.pr, review)
	return err
}

func (g *GitHubPullRequest) postCommentsForEach() error {
	var eg errgroup.Group
	for _, c := range g.postComments {
		comment := c
		if g.postedcs.IsPosted(comment) {
			continue
		}
		eg.Go(func() error {
			body := commentBody(comment)
			prcomment := &github.PullRequestComment{
				CommitID: &g.sha,
				Body:     &body,
				Path:     &comment.Path,
				Position: &comment.LnumDiff,
			}
			_, _, err := g.cli.PullRequests.CreateComment(g.ctx, g.owner, g.repo, g.pr, prcomment)
			return err
		})
	}
	return eg.Wait()
}

func (g *GitHubPullRequest) setPostedComment() error {
	g.postedcs = make(postedcomments)
	cs, err := g.comment()
	if err != nil {
		return err
	}
	for _, c := range cs {
		if c.Position == nil || c.Path == nil || c.Body == nil {
			// skip resolved comments. Or comments which do not have "path" nor
			// "body".
			continue
		}
		path := *c.Path
		pos := *c.Position
		body := *c.Body
		if _, ok := g.postedcs[path]; !ok {
			g.postedcs[path] = make(map[int][]string)
		}
		if _, ok := g.postedcs[path][pos]; !ok {
			g.postedcs[path][pos] = make([]string, 0)
		}
		g.postedcs[path][pos] = append(g.postedcs[path][pos], body)
	}
	return nil
}

// Diff returns a diff of PullRequest. It runs `git diff` locally instead of
// diff_url of GitHub Pull Request because diff of diff_url is not suited for
// comment API in a sense that diff of diff_url is equivalent to
// `git diff --no-renames`, we want diff which is equivalent to
// `git diff --find-renames`.
func (g *GitHubPullRequest) Diff() ([]byte, error) {
	pr, _, err := g.cli.PullRequests.Get(g.ctx, g.owner, g.repo, g.pr)
	if err != nil {
		return nil, err
	}
	b, err := exec.Command("git", "merge-base", g.sha, *pr.Base.SHA).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge-base commit: %v", err)
	}
	mergeBase := strings.Trim(string(b), "\n")
	return exec.Command("git", "diff", "--find-renames", mergeBase, g.sha).Output()
}

// Strip returns 1 as a strip of git diff.
func (g *GitHubPullRequest) Strip() int {
	return 1
}

func (g *GitHubPullRequest) comment() ([]*github.PullRequestComment, error) {
	comments, _, err := g.cli.PullRequests.ListComments(g.ctx, g.owner, g.repo, g.pr, nil)
	if err != nil {
		return nil, err
	}
	return comments, nil
}
