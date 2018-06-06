package reviewdog

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"os/exec"

	"github.com/xanzy/go-gitlab"
	"github.com/masamotod/reviewdog"
)

var _ CommentService = &GitLabMergeRequest{}
var _ DiffService = &GitLabMergeRequest{}

// GitLabMergeRequest is a comment and diff service for GitLab MergeRequest.
type GitLabMergeRequest struct {
	cli      *gitlab.Client
	pid      string
	mr       int
	baseSHA  string
	startSHA string
	headSHA  string

	muComments   sync.Mutex
	postComments []*Comment

	postedcs postedcomments

	// wd is working directory relative to root of repository.
	wd      string
	diffCmd *reviewdog.DiffCmd
}

// NewGitLabMergeRequest returns a new GitLabMergeRequest service.
// GitLabMergeRequest service needs git command in $PATH.
func NewGitLabMergeRequest(cli *gitlab.Client, pid string, mr int, sha string) (*GitLabMergeRequest, error) {
	workDir, err := gitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("GitLabMergeRequest needs 'git' command: %v", err)
	}

	v, err := latestDiffVersion(cli, pid, mr)
	if err != nil {
		return nil, err
	} else if v == nil {
		return nil, fmt.Errorf("no versions")
	}

	cmd := exec.Command("git", "diff", v.StartCommitSHA+"..."+sha)
	d := reviewdog.NewDiffCmd(cmd, 1)

	return &GitLabMergeRequest{
		cli:      cli,
		pid:      pid,
		mr:       mr,
		baseSHA:  v.BaseCommitSHA,
		startSHA: v.StartCommitSHA,
		headSHA:  sha,
		wd:       workDir,
		diffCmd:  d,
	}, nil
}

func latestDiffVersion(cli *gitlab.Client, pid string, mr int) (*gitlab.MergeRequestDiffVersion, error) {
	versions, _, err := cli.MergeRequests.GetMergeRequestDiffVersions(pid, mr, nil)
	if err != nil {
		return nil, err
	}

	var latest *gitlab.MergeRequestDiffVersion = nil
	for _, v := range versions {
		if latest == nil || v.CreatedAt.After(*latest.CreatedAt) {
			latest = v
		}
	}

	return latest, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitLab in parallel.
func (g *GitLabMergeRequest) Post(_ context.Context, c *Comment) error {
	c.Path = filepath.Join(g.wd, c.Path)
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (g *GitLabMergeRequest) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()

	if err := g.setPostedComment(ctx); err != nil {
		return err
	}
	return g.postAsReviewComment(ctx)
}

func (g *GitLabMergeRequest) postAsReviewComment(ctx context.Context) error {
	for _, c := range g.postComments {
		if g.postedcs.IsPosted(c) {
			continue
		}
		cbody := commentBody(c)
		discussion := &gitlab.CreateMergeRequestDiscussionOptions{
			Body: cbody,
			Position: &gitlab.NotePosition{
				BaseSHA:      g.baseSHA,
				StartSHA:     g.startSHA,
				HeadSHA:      g.headSHA,
				PositionType: "text",
				NewPath:      &c.Path,
				NewLine:      &c.LnumDiff,
			},
		}
		_, err := g.cli.Discussions.CreateMergeRequestDiscussion(g.pid, g.mr, discussion)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GitLabMergeRequest) setPostedComment(ctx context.Context) error {
	g.postedcs = make(postedcomments)

	comments, err := g.comment(ctx)
	if err != nil {
		return err
	}

	for _, c := range comments {
		if c.Position == nil ||
			(c.Position != nil && c.Position.NewPath == nil) ||
			(c.Position != nil && c.Position.NewLine == nil) ||
			c.Resolved || c.Body == "" {
			// skip resolved comments. Or comments which do not have "position" nor
			// "body".
			continue
		}

		path := *c.Position.NewPath
		line := *c.Position.NewLine
		body := c.Body
		if _, ok := g.postedcs[path]; !ok {
			g.postedcs[path] = make(map[int][]string)
		}
		if _, ok := g.postedcs[path][line]; !ok {
			g.postedcs[path][line] = make([]string, 0)
		}
		g.postedcs[path][line] = append(g.postedcs[path][line], body)
	}

	return nil
}

// Diff returns a diff of MergeRequest using local git command.
func (g *GitLabMergeRequest) Diff(ctx context.Context) ([]byte, error) {
	return g.diffCmd.Diff(ctx)
}

func (g *GitLabMergeRequest) Strip() int {
	return g.diffCmd.Strip()
}

func (g *GitLabMergeRequest) comment(ctx context.Context) ([]*gitlab.Note, error) {
	comments, _, err := g.cli.Notes.ListMergeRequestNotes(g.pid, g.mr, nil)
	if err != nil {
		return nil, err
	}

	return comments, nil
}
