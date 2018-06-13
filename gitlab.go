package reviewdog

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"
)

var _ CommentService = &GitLabMergeRequest{}
var _ DiffService = &GitLabMergeRequest{}

// GitLabMergeRequest is a comment and diff service for GitLab MergeRequest.
//
// API:
//  https://docs.gitlab.com/ce/api/commits.html#post-comment-to-commit
//  POST /projects/:id/repository/commits/:sha/comments
type GitLabMergeRequest struct {
	cli      *gitlab.Client
	owner    string
	repo     string
	pr       int
	sha      string
	projects string

	muComments   sync.Mutex
	postComments []*Comment

	postedcs postedcomments

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitLabMergeReqest returns a new GitLabMergeRequest service.
// GitLabMergeRequest service needs git command in $PATH.
func NewGitLabMergeReqest(cli *gitlab.Client, owner, repo string, pr int, sha string) (*GitLabMergeRequest, error) {
	workDir, err := gitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("GitLabMergeRequest needs 'git' command: %v", err)
	}
	return &GitLabMergeRequest{
		cli:      cli,
		owner:    owner,
		repo:     repo,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
		wd:       workDir,
	}, nil
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
	defer g.muComments.Unlock()
	g.muComments.Lock()

	if err := g.setPostedComment(ctx); err != nil {
		return err
	}

	return g.postCommentsForEach(ctx)
}

func (g *GitLabMergeRequest) postCommentsForEach(ctx context.Context) error {
	var eg errgroup.Group
	for _, c := range g.postComments {
		comment := c
		if g.postedcs.IsPosted(comment, comment.Lnum) {
			continue
		}
		eg.Go(func() error {
			commitID, err := g.getLastCommitsID(comment.Path, comment.Lnum)
			if err != nil {
				commitID = g.sha
			}
			body := commentBody(comment)
			ltype := "new"
			prcomment := &gitlab.PostCommitCommentOptions{
				Note:     &body,
				Path:     &comment.Path,
				Line:     &comment.Lnum,
				LineType: &ltype,
			}
			_, _, err = g.cli.Commits.PostCommitComment(g.projects, commitID, prcomment, nil)
			return err
		})
	}
	return eg.Wait()
}

func (g *GitLabMergeRequest) getLastCommitsID(path string, line int) (string, error) {
	lineFormat := fmt.Sprintf("%d,%d", line, line)
	s, err := exec.Command("git", "blame", "-l", "-L", lineFormat, path).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commitID: %v", err)
	}
	commitID := strings.Split(string(s), " ")[0]
	return commitID, nil
}

func (g *GitLabMergeRequest) setPostedComment(ctx context.Context) error {
	g.postedcs = make(postedcomments)
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
		path := c.Path
		pos := c.Line
		body := c.Note
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

// Diff returns a diff of MergeRequest. It runs `git diff` locally instead of
// diff_url of GitLab Merge Request because diff of diff_url is not suited for
// comment API in a sense that diff of diff_url is equivalent to
// `git diff --no-renames`, we want diff which is equivalent to
// `git diff --find-renames`.
func (g *GitLabMergeRequest) Diff(ctx context.Context) ([]byte, error) {
	mr, _, err := g.cli.MergeRequests.GetMergeRequest(g.projects, g.pr, nil)
	if err != nil {
		return nil, err
	}
	targetBranch, _, err := g.cli.Branches.GetBranch(mr.TargetProjectID, mr.TargetBranch, nil)
	if err != nil {
		return nil, err
	}
	return g.gitDiff(ctx, g.sha, targetBranch.Commit.ID)
}

func (g *GitLabMergeRequest) gitDiff(ctx context.Context, baseSha string, targetSha string) ([]byte, error) {
	b, err := exec.Command("git", "merge-base", targetSha, baseSha).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge-base commit: %v", err)
	}
	mergeBase := strings.Trim(string(b), "\n")
	relArg := fmt.Sprintf("--relative=%s", g.wd)
	bytes, err := exec.Command("git", "diff", relArg, "--find-renames", mergeBase, baseSha).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %v", err)
	}
	return bytes, nil
}

// Strip returns 1 as a strip of git diff.
func (g *GitLabMergeRequest) Strip() int {
	return 1
}

func (g *GitLabMergeRequest) comment(ctx context.Context) ([]*gitlab.CommitComment, error) {
	commits, _, err := g.cli.MergeRequests.GetMergeRequestCommits(g.projects, g.pr, nil)
	if err != nil {
		return nil, err
	}
	comments := make([]*gitlab.CommitComment, 0)
	for _, c := range commits {
		tmpComments, _, err := g.cli.Commits.GetCommitComments(g.projects, c.ID, nil)
		if err != nil {
			continue
		}
		comments = append(comments, tmpComments...)
	}
	return comments, nil
}
