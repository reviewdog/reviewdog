package reviewdog

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

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
	postComments []*Comment

	cli   *github.Client
	owner string
	repo  string
	pr    int
	sha   string

	postedcs postedcomments

	muFlash sync.Mutex
}

// NewGitHubPullReqest returns a new GitHubPullRequest service.
func NewGitHubPullReqest(cli *github.Client, owner, repo string, pr int, sha string) *GitHubPullRequest {
	return &GitHubPullRequest{
		cli:   cli,
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
}

func F() {
	go func() {
		var tick = time.Tick(1 * time.Second)
		for t := range tick {
			fmt.Println(t)
		}
	}()
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
	comments := make([]*ReviewComment, 0, len(g.postComments))
	for _, c := range g.postComments {
		if g.postedcs.IsPosted(c) {
			continue
		}
		cbody := commentBody(c)
		comments = append(comments, &ReviewComment{
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
	event := "COMMENT"
	review := &Review{Event: &event, Comments: comments}

	_, _, err := g.CreateReview(g.owner, g.repo, g.pr, review)
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
			_, _, err := g.cli.PullRequests.CreateComment(g.owner, g.repo, g.pr, prcomment)
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
	pr, _, err := g.cli.PullRequests.Get(g.owner, g.repo, g.pr)
	if err != nil {
		return nil, err
	}
	return exec.Command("git", "diff", "--find-renames", *pr.Base.SHA, g.sha).Output()
}

// Strip returns 1 as a strip of git diff.
func (g *GitHubPullRequest) Strip() int {
	return 1
}

func (g *GitHubPullRequest) comment() ([]*github.PullRequestComment, error) {
	comments, _, err := g.cli.PullRequests.ListComments(g.owner, g.repo, g.pr, nil)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// ---
// GitHub PullRequest Review API Implementation
// ref: https://github.com/google/go-github/issues/495

const (
	mediaTypePullRequestReview = "application/vnd.github.black-cat-preview+json"
)

// Review represents a pull request review.
type Review struct {
	Body     *string          `json:"body,omitempty"`
	Event    *string          `json:"event,omitempty"`
	Comments []*ReviewComment `json:"comments,omitempty"`
}

// ReviewComment represents draft review comments.
type ReviewComment struct {
	Path     *string `json:"path,omitempty"`
	Position *int    `json:"position,omitempty"`
	Body     *string `json:"body,omitempty"`
}

// CreateReview creates a new review comment on the specified pull request.
//
// GitHub API docs: https://developer.github.com/v3/pulls/reviews/#create-a-pull-request-review
func (g *GitHubPullRequest) CreateReview(owner, repo string, number int, review *Review) (*github.PullRequestReview, *github.Response, error) {
	u := fmt.Sprintf("repos/%v/%v/pulls/%d/reviews", owner, repo, number)

	req, err := g.cli.NewRequest("POST", u, review)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Accept", mediaTypePullRequestReview)

	r := new(github.PullRequestReview)
	resp, err := g.cli.Do(req, r)
	if err != nil {
		log.Printf("GitHub Review API error: %v", err)
		return nil, resp, err
	}
	return r, resp, err
}
