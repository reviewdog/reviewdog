package watchdogs

import (
	"bytes"

	"github.com/google/go-github/github"
	"golang.org/x/sync/errgroup"
)

var _ = github.ScopeAdminOrg

var _ CommentService = &GitHubPullRequest{}
var _ DiffService = &GitHubPullRequest{}

// `path` to `position`(LnumDiff) to comment `body`s
type postedcomments map[string]map[int][]string

// IsPosted returns true if a given comment has been posted in GitHub already,
// otherwise returns false. It sees comments with same path, same position,
// and same body as same comments.
func (p postedcomments) IsPosted(c *Comment) bool {
	if _, ok := p[c.Path]; !ok {
		return false
	}
	if _, ok := p[c.Path][c.LnumDiff]; !ok {
		return false
	}
	return true
}

// https://developer.github.com/v3/pulls/comments/#create-a-comment
// POST /repos/:owner/:repo/pulls/:number/comments
type GitHubPullRequest struct {
	postComments []*Comment

	cli   *github.Client
	owner string
	repo  string
	pr    int
	sha   string

	postedcs postedcomments
}

func NewGitHubPullReqest(cli *github.Client, owner, repo string, pr int, sha string) *GitHubPullRequest {
	return &GitHubPullRequest{
		cli:   cli,
		owner: owner,
		repo:  repo,
		pr:    pr,
		sha:   sha,
	}
}

// Post accepts a comment and holds it. Flash method actually posts comments to
// GitHub in parallel.
func (g *GitHubPullRequest) Post(c *Comment) error {
	g.postComments = append(g.postComments, c)
	return nil
}

// ListPostComments lists comments to post.
func (g *GitHubPullRequest) ListPostComments() []*Comment {
	return g.postComments
}

// Flash posts comments which has not been posted yet.
func (g *GitHubPullRequest) Flash() error {
	if err := g.setPostedComment(); err != nil {
		return err
	}
	var eg errgroup.Group
	for _, c := range g.ListPostComments() {
		comment := c
		if g.postedcs.IsPosted(comment) {
			continue
		}
		eg.Go(func() error {
			prcomment := &github.PullRequestComment{
				CommitID: &g.sha,
				Body:     &comment.Body,
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

func (g *GitHubPullRequest) Diff() ([]byte, error) {
	pr, _, err := g.cli.PullRequests.Get(g.owner, g.repo, g.pr)
	if err != nil {
		return nil, err
	}
	req, err := g.cli.NewRequest("GET", *pr.DiffURL, nil)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := g.cli.Do(req, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *GitHubPullRequest) Strip() int {
	return 1
}

func (g *GitHubPullRequest) comment() ([]github.PullRequestComment, error) {
	comments, _, err := g.cli.PullRequests.ListComments(g.owner, g.repo, g.pr, nil)
	if err != nil {
		return nil, err
	}
	return comments, nil
}
