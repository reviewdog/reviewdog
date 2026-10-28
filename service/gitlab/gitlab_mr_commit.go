package gitlab

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/commentutil"
)

var _ reviewdog.CommentService = &MergeRequestCommitCommenter{}

// MergeRequestCommitCommenter is a comment service for GitLab MergeRequest.
//
// API:
//
//	https://docs.gitlab.com/ce/api/commits.html#post-comment-to-commit
//	POST /projects/:id/repository/commits/:sha/comments
type MergeRequestCommitCommenter struct {
	cli      *gitlab.Client
	pr       int
	sha      string
	projects string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	postedcs commentutil.PostedComments
}

// NewGitLabMergeRequestCommitCommenter returns a new MergeRequestCommitCommenter service.
// MergeRequestCommitCommenter service needs git command in $PATH.
func NewGitLabMergeRequestCommitCommenter(cli *gitlab.Client, owner, repo string, pr int, sha string) *MergeRequestCommitCommenter {
	return &MergeRequestCommitCommenter{
		cli:      cli,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
	}
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitLab in parallel.
func (g *MergeRequestCommitCommenter) Post(_ context.Context, c *reviewdog.Comment) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

func (*MergeRequestCommitCommenter) ShouldPrependGitRelDir() bool { return true }

// Flush posts comments which has not been posted yet.
func (g *MergeRequestCommitCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	defer func() { g.postComments = nil }()

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
		if !c.Result.InDiffFile || lnum == 0 {
			continue
		}
		eg.Go(func() error {
			commitID, line := g.sha, lnum
			if id, l, err := g.getLastCommitsID(loc.GetPath(), lnum); err == nil {
				commitID, line = id, l
			}
			if g.postedcs.IsPosted(c, line, body) {
				return nil
			}
			prcomment := &gitlab.PostCommitCommentOptions{
				Note:     new(body),
				Path:     new(loc.GetPath()),
				Line:     new(int64(line)),
				LineType: new("new"),
			}
			_, _, err := g.cli.Commits.PostCommitComment(g.projects, commitID, prcomment, gitlab.WithContext(ctx))
			return err
		})
	}
	return eg.Wait()
}

// getLastCommitsID returns the commit that last changed the given line and the
// line number in that commit, which differs from line when a later commit
// added or removed lines above it.
func (g *MergeRequestCommitCommenter) getLastCommitsID(path string, line int) (string, int, error) {
	lineFormat := fmt.Sprintf("%d,%d", line, line)
	s, err := exec.Command("git", "blame", "--porcelain", "-L", lineFormat, path).Output()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get commitID: %w", err)
	}
	// The first line is "<commit> <original line> <final line> <lines>". Unlike
	// the default format, the commit is never prefixed with "^" for boundary
	// commits (e.g. the oldest commit of a shallow clone).
	header, _, _ := strings.Cut(string(s), "\n")
	fields := strings.Fields(header)
	if len(fields) < 3 {
		return "", 0, fmt.Errorf("unexpected git blame output: %q", header)
	}
	commitID := fields[0]
	if strings.Trim(commitID, "0") == "" {
		return "", 0, fmt.Errorf("line %d of %s is not committed yet", line, path)
	}
	origLine, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", 0, fmt.Errorf("unexpected git blame output: %q", header)
	}
	return commitID, origLine, nil
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
		g.postedcs.AddPostedComment(c.Path, int(c.Line), c.Note)
	}
	return nil
}

func (g *MergeRequestCommitCommenter) comment(ctx context.Context) ([]*gitlab.CommitComment, error) {
	commits, _, err := g.cli.MergeRequests.GetMergeRequestCommits(
		g.projects, int64(g.pr), nil, gitlab.WithContext(ctx))
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
