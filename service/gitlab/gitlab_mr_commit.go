package gitlab

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"slices"
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
	// commits are the merge request's commits and the head commit, which are
	// the ones comments are posted on.
	commits map[string]bool
}

// postedComment identifies a comment by the commit, path and line it's posted
// on, and its body. An empty commit matches a comment on any commit.
type postedComment struct {
	commit string
	path   string
	line   int
	body   string
}

// errNotCommitted is returned by getLastCommitsID for lines that only exist in
// the working tree.
var errNotCommitted = errors.New("line is not committed yet")

// errBoundary is returned by getLastCommitsID for lines that git blame
// attributes to a boundary commit, e.g. the oldest commit of a shallow clone.
// Such lines may come from an older commit that isn't in the clone.
var errBoundary = errors.New("line is attributed to a boundary commit")

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
			id, p, l, err := g.getLastCommitsID(ctx, loc.GetPath(), lnum)
			if errors.Is(err, errNotCommitted) {
				// No commit has the line, and the same line number in the head
				// commit may be another line or past the end of the file.
				log.Printf("reviewdog: skipping comment on %s:%d: %v", loc.GetPath(), lnum, err)
				return nil
			}
			commitID, path, line := g.sha, loc.GetPath(), lnum
			// Comment on the head commit when the line comes from a commit outside
			// the merge request, e.g. on the target branch, since only comments on
			// the merge request's commits are checked for duplicates. The head
			// changes on every push, so the comment counts as posted if any of
			// those commits already has it at the same position.
			posted := postedComment{path: path, line: line, body: body}
			if err == nil && g.commits[id] {
				commitID, path, line = id, p, l
				posted = postedComment{commit: id, path: p, line: l, body: body}
			}
			if g.postedcs[posted] {
				return nil
			}
			prcomment := &gitlab.PostCommitCommentOptions{
				Note:     new(body),
				Path:     new(path),
				Line:     new(int64(line)),
				LineType: new("new"),
			}
			_, _, err = g.cli.Commits.PostCommitComment(g.projects, commitID, prcomment, gitlab.WithContext(ctx))
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
		return "", "", 0, errNotCommitted
	}
	origLine, err = strconv.Atoi(fields[1])
	if err != nil {
		return "", "", 0, fmt.Errorf("unexpected git blame output: %q", lines[0])
	}
	// The header is followed by "<key> <value>" lines and then by the content
	// of the line, which starts with a tab. "filename" is the file's path in
	// the commit, and "boundary" marks a boundary commit.
	origPath = path
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "\t") {
			break
		}
		if l == "boundary" {
			return "", "", 0, errBoundary
		}
		if name, ok := strings.CutPrefix(l, "filename "); ok {
			// Paths with unusual characters are quoted, like in other git output.
			if strings.HasPrefix(name, `"`) {
				if unquoted, err := strconv.Unquote(name); err == nil {
					name = unquoted
				}
			}
			origPath = name
		}
	}
	return commitID, origPath, origLine, nil
}

func (g *MergeRequestCommitCommenter) setPostedComment(ctx context.Context) error {
	g.postedcs = make(map[postedComment]bool)
	g.commits = make(map[string]bool)
	commits, err := g.listAllMergeRequestCommits(ctx)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(commits)+1)
	for _, commit := range commits {
		ids = append(ids, commit.ID)
	}
	// In merged results pipelines, the head commit is a merge commit that
	// isn't one of the merge request's commits.
	if !slices.Contains(ids, g.sha) {
		ids = append(ids, g.sha)
	}
	for _, id := range ids {
		// Comments are posted on these commits, so a commit whose comments
		// can't be listed would get them posted again.
		cs, err := g.listAllCommitComments(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to list comments on commit %s: %w", id, err)
		}
		g.commits[id] = true
		for _, c := range cs {
			if c.Line == 0 || c.Path == "" || c.Note == "" {
				// skip resolved comments. Or comments which do not have "path" nor
				// "body".
				continue
			}
			g.postedcs[postedComment{commit: id, path: c.Path, line: int(c.Line), body: c.Note}] = true
			g.postedcs[postedComment{path: c.Path, line: int(c.Line), body: c.Note}] = true
		}
	}
	return nil
}

func (g *MergeRequestCommitCommenter) listAllMergeRequestCommits(ctx context.Context) ([]*gitlab.Commit, error) {
	opts := &gitlab.GetMergeRequestCommitsOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
	var all []*gitlab.Commit
	for {
		commits, resp, err := g.cli.MergeRequests.GetMergeRequestCommits(
			g.projects, int64(g.pr), opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		all = append(all, commits...)
		if resp.NextPage == 0 {
			return all, nil
		}
		opts.Page = resp.NextPage
	}
}

func (g *MergeRequestCommitCommenter) listAllCommitComments(ctx context.Context, sha string) ([]*gitlab.CommitComment, error) {
	opts := &gitlab.GetCommitCommentsOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
	var all []*gitlab.CommitComment
	for {
		cs, resp, err := g.cli.Commits.GetCommitComments(
			g.projects, sha, opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		all = append(all, cs...)
		if resp.NextPage == 0 {
			return all, nil
		}
		opts.Page = resp.NextPage
	}
}
