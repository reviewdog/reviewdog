package reviewdog

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"
)

// GitLabMergeRequestDiscussionCommenter is a comment and diff service for GitLab MergeRequest.
//
// API:
//  https://docs.gitlab.com/ee/api/discussions.html#create-new-merge-request-discussion
//  POST /projects/:id/merge_requests/:merge_request_iid/discussions
type GitLabMergeRequestDiscussionCommenter struct {
	cli      *gitlab.Client
	pr       int
	sha      string
	projects string

	muComments   sync.Mutex
	postComments []*Comment

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitLabMergeRequestDiscussionCommenter returns a new GitLabMergeRequestDiscussionCommenter service.
// GitLabMergeRequestDiscussionCommenter service needs git command in $PATH.
func NewGitLabMergeRequestDiscussionCommenter(cli *gitlab.Client, owner, repo string, pr int, sha string) (*GitLabMergeRequestDiscussionCommenter, error) {
	workDir, err := gitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("GitLabMergeRequestDiscussionCommenter needs 'git' command: %v", err)
	}
	return &GitLabMergeRequestDiscussionCommenter{
		cli:      cli,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
		wd:       workDir,
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitLab in parallel.
func (g *GitLabMergeRequestDiscussionCommenter) Post(_ context.Context, c *Comment) error {
	c.Path = filepath.Join(g.wd, c.Path)
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (g *GitLabMergeRequestDiscussionCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	postedcs, err := g.createPostedCommetns()
	if err != nil {
		return fmt.Errorf("failed to create posted comments: %v", err)
	}
	return g.postCommentsForEach(ctx, postedcs)
}

func (g *GitLabMergeRequestDiscussionCommenter) createPostedCommetns() (postedcomments, error) {
	postedcs := make(postedcomments)
	discussions, err := listAllMergeRequestDiscussion(g.cli, g.projects, g.pr, &ListMergeRequestDiscussionOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list all merge request discussions: %v", err)
	}
	for _, d := range discussions {
		for _, note := range d.Notes {
			pos := note.Position
			if pos == nil || pos.NewPath == "" || pos.NewLine == 0 || note.Body == "" {
				continue
			}
			postedcs.AddPostedComment(pos.NewPath, pos.NewLine, note.Body)
		}
	}
	return postedcs, nil
}

func (g *GitLabMergeRequestDiscussionCommenter) postCommentsForEach(ctx context.Context, postedcs postedcomments) error {
	mr, _, err := g.cli.MergeRequests.GetMergeRequest(g.projects, g.pr, nil, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get merge request: %v", err)
	}
	targetBranch, _, err := g.cli.Branches.GetBranch(mr.TargetProjectID, mr.TargetBranch, nil)
	if err != nil {
		return err
	}

	var eg errgroup.Group
	for _, c := range g.postComments {
		comment := c
		if postedcs.IsPosted(comment, comment.Lnum) {
			continue
		}
		eg.Go(func() error {
			discussion := &GitLabMergeRequestDiscussion{
				Body: commentBody(comment),
				Position: &GitLabMergeRequestDiscussionPosition{
					StartSHA:     targetBranch.Commit.ID,
					HeadSHA:      g.sha,
					BaseSHA:      g.sha,
					PositionType: "text",
					NewPath:      comment.Path,
					NewLine:      comment.Lnum,
				},
			}
			_, err := CreateMergeRequestDiscussion(g.cli, g.projects, g.pr, discussion)
			return err
		})
	}
	return eg.Wait()
}

// GitLabMergeRequestDiscussionPosition represents position of GitLab MergeRequest Discussion.
type GitLabMergeRequestDiscussionPosition struct {
	// Required.
	BaseSHA      string `json:"base_sha,omitempty"`      // Base commit SHA in the source branch
	StartSHA     string `json:"start_sha,omitempty"`     // SHA referencing commit in target branch
	HeadSHA      string `json:"head_sha,omitempty"`      // SHA referencing HEAD of this merge request
	PositionType string `json:"position_type,omitempty"` // Type of the position reference', allowed values: 'text' or 'image'

	// Optional.
	NewPath string `json:"new_path,omitempty"` // File path after change
	NewLine int    `json:"new_line,omitempty"` // Line number after change (for 'text' diff notes)
	OldPath string `json:"old_path,omitempty"` // File path before change
	OldLine int    `json:"old_line,omitempty"` // Line number before change (for 'text' diff notes)
}

// GitLabMergeRequestDiscussionList represents response of ListMergeRequestDiscussion API.
//
// GitLab API docs: https://docs.gitlab.com/ee/api/discussions.html#list-project-merge-request-discussions
type GitLabMergeRequestDiscussionList struct {
	Notes []*GitLabMergeRequestDiscussion `json:"notes"`
}

// GitLabMergeRequestDiscussion represents a discussion of MergeRequest.
type GitLabMergeRequestDiscussion struct {
	Body     string                                `json:"body"` // The content of a discussion
	Position *GitLabMergeRequestDiscussionPosition `json:"position"`
}

// CreateMergeRequestDiscussion creates new discussion on a merge request.
//
// GitLab API docs: https://docs.gitlab.com/ee/api/discussions.html#create-new-merge-request-discussion
func CreateMergeRequestDiscussion(cli *gitlab.Client, projectID string, mergeRequest int, discussion *GitLabMergeRequestDiscussion) (*gitlab.Response, error) {
	u := fmt.Sprintf("projects/%s/merge_requests/%d/discussions", url.QueryEscape(projectID), mergeRequest)
	req, err := cli.NewRequest("POST", u, discussion, nil)
	if err != nil {
		return nil, err
	}
	return cli.Do(req, nil)
}

// ListMergeRequestDiscussion lists discussion on a merge request.
//
// GitLab API docs: https://docs.gitlab.com/ee/api/discussions.html#list-project-merge-request-discussions
func ListMergeRequestDiscussion(cli *gitlab.Client, projectID string, mergeRequest int, opts *ListMergeRequestDiscussionOptions) ([]*GitLabMergeRequestDiscussionList, *gitlab.Response, error) {
	u := fmt.Sprintf("projects/%s/merge_requests/%d/discussions", url.QueryEscape(projectID), mergeRequest)
	req, err := cli.NewRequest("GET", u, opts, nil)
	if err != nil {
		return nil, nil, err
	}
	var discussions []*GitLabMergeRequestDiscussionList
	resp, err := cli.Do(req, &discussions)
	if err != nil {
		return nil, resp, err
	}
	return discussions, resp, nil
}

// ListMergeRequestDiscussionOptions represents the available ListMergeRequestDiscussion() options.
//
// GitLab API docs: https://docs.gitlab.com/ee/api/discussions.html#list-project-merge-request-discussions
type ListMergeRequestDiscussionOptions gitlab.ListOptions

func listAllMergeRequestDiscussion(cli *gitlab.Client, projectID string, mergeRequest int, opts *ListMergeRequestDiscussionOptions) ([]*GitLabMergeRequestDiscussionList, error) {
	discussions, resp, err := ListMergeRequestDiscussion(cli, projectID, mergeRequest, opts)
	if err != nil {
		return nil, err
	}
	if resp.NextPage == 0 {
		return discussions, nil
	}
	newOpts := &ListMergeRequestDiscussionOptions{
		Page:    resp.NextPage,
		PerPage: opts.PerPage,
	}
	restDiscussions, err := listAllMergeRequestDiscussion(cli, projectID, mergeRequest, newOpts)
	if err != nil {
		return nil, err
	}
	return append(discussions, restDiscussions...), nil
}
