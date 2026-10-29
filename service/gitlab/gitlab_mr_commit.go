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

	postedcs map[postedComment]bool
}

// postedComment identifies a comment by the commit, path and line it's posted
// on, and its body.
type postedComment struct {
	commit string
	path   string
	line   int
	body   string
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
			commitID, path, line := g.sha, loc.GetPath(), lnum
			if id, p, l, err := g.getLastCommitsID(ctx, loc.GetPath(), lnum); err == nil {
				commitID, path, line = id, p, l
			}
			if g.postedcs[postedComment{commit: commitID, path: path, line: line, body: body}] {
				return nil
			}
			prcomment := &gitlab.PostCommitCommentOptions{
				Note:     new(body),
				Path:     new(path),
				Line:     new(int64(line)),
				LineType: new("new"),
			}
			_, _, err := g.cli.Commits.PostCommitComment(g.projects, commitID, prcomment, gitlab.WithContext(ctx))
			return err
		})
	}
	return eg.Wait()
}

// getLastCommitsID returns the commit that last changed the given line, and
// the file's path and the line number in that commit. They differ from path
// and line when a later commit renamed the file or added or removed lines
// above it.
func (g *MergeRequestCommitCommenter) getLastCommitsID(ctx context.Context, path string, line int) (commitID, origPath string, origLine int, err error) {
	lineFormat := fmt.Sprintf("%d,%d", line, line)
	s, err := exec.CommandContext(ctx, "git", "blame", "--porcelain", "-L", lineFormat, path).Output()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get commitID: %w", err)
	}
	// The first line is "<commit> <original line> <final line> <lines>". Unlike
	// the default format, the commit is never prefixed with "^" for boundary
	// commits (e.g. the oldest commit of a shallow clone).
	lines := strings.Split(string(s), "\n")
	fields := strings.Fields(lines[0])
	if len(fields) < 3 {
		return "", "", 0, fmt.Errorf("unexpected git blame output: %q", lines[0])
	}
	commitID = fields[0]
	if strings.Trim(commitID, "0") == "" {
		return "", "", 0, fmt.Errorf("line %d of %s is not committed yet", line, path)
	}
	origLine, err = strconv.Atoi(fields[1])
	if err != nil {
		return "", "", 0, fmt.Errorf("unexpected git blame output: %q", lines[0])
	}
	// The header is followed by "<key> <value>" lines and then by the content
	// of the line, which starts with a tab. "filename" is the file's path in
	// the commit.
	origPath = path
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "\t") {
			break
		}
		if name, ok := strings.CutPrefix(l, "filename "); ok {
			// Paths with unusual characters are quoted, like in other git output.
			if strings.HasPrefix(name, `"`) {
				if unquoted, err := strconv.Unquote(name); err == nil {
					name = unquoted
				}
			}
			origPath = name
			break
		}
	}
	return commitID, origPath, origLine, nil
}

func (g *MergeRequestCommitCommenter) setPostedComment(ctx context.Context) error {
	g.postedcs = make(map[postedComment]bool)
	commits, _, err := g.cli.MergeRequests.GetMergeRequestCommits(
		g.projects, int64(g.pr), nil, gitlab.WithContext(ctx))
	if err != nil {
		return err
	}
	for _, commit := range commits {
		cs, _, err := g.cli.Commits.GetCommitComments(
			g.projects, commit.ID, nil, gitlab.WithContext(ctx))
		if err != nil {
			continue
		}
		for _, c := range cs {
			if c.Line == 0 || c.Path == "" || c.Note == "" {
				// skip resolved comments. Or comments which do not have "path" nor
				// "body".
				continue
			}
			g.postedcs[postedComment{commit: commit.ID, path: c.Path, line: int(c.Line), body: c.Note}] = true
		}
	}
	return nil
}
