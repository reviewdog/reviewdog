package reviewdog

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/github"
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
	cli   *github.Client
	owner string
	repo  string
	pr    int
	sha   string

	muComments   sync.Mutex
	postComments []*Comment

	postedcs postedcomments

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitHubPullReqest returns a new GitHubPullRequest service.
// GitHubPullRequest service needs git command in $PATH.
func NewGitHubPullReqest(cli *github.Client, owner, repo string, pr int, sha string) (*GitHubPullRequest, error) {
	workDir, err := gitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("GitHubPullRequest needs 'git' command: %v", err)
	}
	return &GitHubPullRequest{
		cli:   cli,
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
		wd:    workDir,
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitHub in parallel.
func (g *GitHubPullRequest) Post(_ context.Context, c *Comment) error {
	c.Path = filepath.Join(g.wd, c.Path)
	g.muComments.Lock()
	defer g.muComments.Unlock()
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

// Flush posts comments which has not been posted yet.
func (g *GitHubPullRequest) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()

	if err := g.setPostedComment(ctx); err != nil {
		return err
	}
	return g.postAsReviewComment(ctx)
}

func (g *GitHubPullRequest) postAsReviewComment(ctx context.Context) error {
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
		CommitID: &g.sha,
		Event:    github.String("COMMENT"),
		Comments: comments,
	}
	_, _, err := g.cli.PullRequests.CreateReview(ctx, g.owner, g.repo, g.pr, review)
	return err
}

func (g *GitHubPullRequest) setPostedComment(ctx context.Context) error {
	g.postedcs = make(postedcomments)
	cs, err := g.comment(ctx)
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
func (g *GitHubPullRequest) Diff(ctx context.Context) ([]byte, error) {
	pr, _, err := g.cli.PullRequests.Get(ctx, g.owner, g.repo, g.pr)
	if err != nil {
		return nil, err
	}
	return g.gitDiff(ctx, *pr.Base.SHA)
}

func (g *GitHubPullRequest) gitDiff(ctx context.Context, baseSha string) ([]byte, error) {
	b, err := exec.Command("git", "merge-base", g.sha, baseSha).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge-base commit: %v", err)
	}
	mergeBase := strings.Trim(string(b), "\n")
	relArg := fmt.Sprintf("--relative=%s", g.wd)
	bytes, err := exec.Command("git", "diff", relArg, "--find-renames", mergeBase, g.sha).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %v", err)
	}
	return bytes, nil
}

// Strip returns 1 as a strip of git diff.
func (g *GitHubPullRequest) Strip() int {
	return 1
}

func (g *GitHubPullRequest) comment(ctx context.Context) ([]*github.PullRequestComment, error) {
	comments, _, err := g.cli.PullRequests.ListComments(ctx, g.owner, g.repo, g.pr, nil)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func gitRelWorkdir() (string, error) {
	b, err := exec.Command("git", "rev-parse", "--show-prefix").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'git rev-parse --show-prefix': %v", err)
	}
	return strings.Trim(string(b), "\n"), nil
}
