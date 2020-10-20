package gitlab

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/commentutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ reviewdog.CommentService = &MergeRequestCommitCommenter{}

// MergeRequestCommitCommenter is a comment service for GitLab MergeRequest.
//
// API:
//  https://docs.gitlab.com/ce/api/commits.html#post-comment-to-commit
//  POST /projects/:id/repository/commits/:sha/comments
type MergeRequestCommitCommenter struct {
	cli      *gitlab.Client
	pr       int
	sha      string
	projects string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	postedcs commentutil.PostedComments

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitLabMergeRequestCommitCommenter returns a new MergeRequestCommitCommenter service.
// MergeRequestCommitCommenter service needs git command in $PATH.
func NewGitLabMergeRequestCommitCommenter(cli *gitlab.Client, owner, repo string, pr int, sha string) (*MergeRequestCommitCommenter, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("MergeRequestCommitCommenter needs 'git' command: %w", err)
	}
	return &MergeRequestCommitCommenter{
		cli:      cli,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
		wd:       workDir,
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitLab in parallel.
func (g *MergeRequestCommitCommenter) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(
		filepath.Join(g.wd, c.Result.Diagnostic.GetLocation().GetPath()))
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (g *MergeRequestCommitCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()

	if err := g.setPostedComment(ctx); err != nil {
		return err
	}

	return g.postCommentsForEach(ctx)
}

func (g *MergeRequestCommitCommenter) postCommentsForEach(ctx context.Context) error {
	var eg errgroup.Group
	for _, c := range g.postComments {
		c := c
		loc := c.Result.Diagnostic.GetLocation()
		lnum := int(loc.GetRange().GetStart().GetLine())
		body := commentutil.MarkdownComment(c)
		if !c.Result.InDiffFile || lnum == 0 || g.postedcs.IsPosted(c, lnum, body) {
			continue
		}
		eg.Go(func() error {
			commitID, err := g.getLastCommitsID(loc.GetPath(), lnum)
			if err != nil {
				commitID = g.sha
			}
			prcomment := &gitlab.PostCommitCommentOptions{
				Note:     gitlab.String(body),
				Path:     gitlab.String(loc.GetPath()),
				Line:     gitlab.Int(lnum),
				LineType: gitlab.String("new"),
			}
			_, _, err = g.cli.Commits.PostCommitComment(g.projects, commitID, prcomment, gitlab.WithContext(ctx))
			return err
		})
	}
	return eg.Wait()
}

func (g *MergeRequestCommitCommenter) getLastCommitsID(path string, line int) (string, error) {
	lineFormat := fmt.Sprintf("%d,%d", line, line)
	s, err := exec.Command("git", "blame", "-l", "-L", lineFormat, path).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commitID: %w", err)
	}
	commitID := strings.Split(string(s), " ")[0]
	return commitID, nil
}

func (g *MergeRequestCommitCommenter) setPostedComment(ctx context.Context) error {
	g.postedcs = make(commentutil.PostedComments)
	cs, err := g.comment(ctx)
	if err != nil {
		return err
	}
	for _, c := range cs {
		if c.Line == 0 || c.Path == "" || c.Note == "" {
			// skip resolved comments. Or comments which do not have "path" nor
			// "body".
			continue
		}
		g.postedcs.AddPostedComment(c.Path, c.Line, c.Note)
	}
	return nil
}

func (g *MergeRequestCommitCommenter) comment(ctx context.Context) ([]*gitlab.CommitComment, error) {
	commits, _, err := g.cli.MergeRequests.GetMergeRequestCommits(
		g.projects, g.pr, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	comments := make([]*gitlab.CommitComment, 0)
	for _, c := range commits {
		tmpComments, _, err := g.cli.Commits.GetCommitComments(
			g.projects, c.ID, nil, gitlab.WithContext(ctx))
		if err != nil {
			continue
		}
		comments = append(comments, tmpComments...)
	}
	return comments, nil
}
