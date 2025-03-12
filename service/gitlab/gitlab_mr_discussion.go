package gitlab

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/commentutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

const (
	invalidSuggestionPre  = "<details><summary>reviewdog suggestion error</summary>"
	invalidSuggestionPost = "</details>"
)

// MergeRequestDiscussionCommenter is a comment and diff service for GitLab MergeRequest.
//
// API:
//
//	https://docs.gitlab.com/ee/api/discussions.html#create-new-merge-request-discussion
//	POST /projects/:id/merge_requests/:merge_request_iid/discussions
type MergeRequestDiscussionCommenter struct {
	cli      *gitlab.Client
	pr       int
	sha      string
	projects string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitLabMergeRequestDiscussionCommenter returns a new MergeRequestDiscussionCommenter service.
// MergeRequestDiscussionCommenter service needs git command in $PATH.
func NewGitLabMergeRequestDiscussionCommenter(cli *gitlab.Client, owner, repo string, pr int, sha string) (*MergeRequestDiscussionCommenter, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("MergeRequestDiscussionCommenter needs 'git' command: %w", err)
	}
	return &MergeRequestDiscussionCommenter{
		cli:      cli,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
		wd:       workDir,
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitLab in parallel.
func (g *MergeRequestDiscussionCommenter) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Diagnostic.GetLocation().Path = filepath.ToSlash(
		filepath.Join(g.wd, c.Result.Diagnostic.GetLocation().GetPath()))
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (g *MergeRequestDiscussionCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	defer func() { g.postComments = nil }()
	postedcs, err := g.createPostedComments()
	if err != nil {
		return fmt.Errorf("failed to create posted comments: %w", err)
	}
	return g.postCommentsForEach(ctx, postedcs)
}

func (g *MergeRequestDiscussionCommenter) createPostedComments() (commentutil.PostedComments, error) {
	postedcs := make(commentutil.PostedComments)
	discussions, err := listAllMergeRequestDiscussion(g.cli, g.projects, g.pr, &gitlab.ListMergeRequestDiscussionsOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list all merge request discussions: %w", err)
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

func (g *MergeRequestDiscussionCommenter) postCommentsForEach(ctx context.Context, postedcs commentutil.PostedComments) error {
	mr, _, err := g.cli.MergeRequests.GetMergeRequest(g.projects, g.pr, nil, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get merge request: %w", err)
	}
	targetBranch, _, err := g.cli.Branches.GetBranch(mr.TargetProjectID, mr.TargetBranch, nil)
	if err != nil {
		return err
	}

	var eg errgroup.Group
	for _, c := range g.postComments {
		c := c
		loc := c.Result.Diagnostic.GetLocation()
		lnum := int(loc.GetRange().GetStart().GetLine())
		body := commentutil.MarkdownComment(c)

		if suggestion := buildSuggestions(c); suggestion != "" {
			body = body + "\n\n" + suggestion
		}

		if !c.Result.InDiffFile || lnum == 0 || postedcs.IsPosted(c, lnum, body) {
			continue
		}
		eg.Go(func() error {
			pos := &gitlab.PositionOptions{
				StartSHA:     gitlab.Ptr(targetBranch.Commit.ID),
				HeadSHA:      gitlab.Ptr(g.sha),
				BaseSHA:      gitlab.Ptr(targetBranch.Commit.ID),
				PositionType: gitlab.Ptr("text"),
				NewPath:      gitlab.Ptr(loc.GetPath()),
				NewLine:      gitlab.Ptr(lnum),
			}
			if c.Result.OldPath != "" && c.Result.OldLine != 0 {
				pos.OldPath = gitlab.Ptr(c.Result.OldPath)
				pos.OldLine = gitlab.Ptr(c.Result.OldLine)
			}
			discussion := &gitlab.CreateMergeRequestDiscussionOptions{
				Body:     gitlab.Ptr(body),
				Position: pos,
			}
			_, _, err := g.cli.Discussions.CreateMergeRequestDiscussion(g.projects, g.pr, discussion)
			if err != nil {
				return fmt.Errorf("failed to create merge request discussion: %w", err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func listAllMergeRequestDiscussion(cli *gitlab.Client, projectID string, mergeRequest int, opts *gitlab.ListMergeRequestDiscussionsOptions) ([]*gitlab.Discussion, error) {
	discussions, resp, err := cli.Discussions.ListMergeRequestDiscussions(projectID, mergeRequest, opts)
	if err != nil {
		return nil, err
	}
	if resp.NextPage == 0 {
		return discussions, nil
	}
	newOpts := &gitlab.ListMergeRequestDiscussionsOptions{
		Page:    resp.NextPage,
		PerPage: opts.PerPage,
	}
	restDiscussions, err := listAllMergeRequestDiscussion(cli, projectID, mergeRequest, newOpts)
	if err != nil {
		return nil, err
	}
	return append(discussions, restDiscussions...), nil
}

// creates diff in markdown for suggested changes
// Ref gitlab suggestion: https://docs.gitlab.com/ee/user/project/merge_requests/reviews/suggestions.html
func buildSuggestions(c *reviewdog.Comment) string {
	var sb strings.Builder
	for _, s := range c.Result.Diagnostic.GetSuggestions() {
		if s.Range == nil || s.Range.Start == nil || s.Range.End == nil {
			continue
		}

		txt, err := buildSingleSuggestion(c, s)
		if err != nil {
			sb.WriteString(invalidSuggestionPre + err.Error() + invalidSuggestionPost + "\n")
			continue
		}
		sb.WriteString(txt)
		sb.WriteString("\n")
	}

	return sb.String()
}

func buildSingleSuggestion(c *reviewdog.Comment, s *rdf.Suggestion) (string, error) {
	var sb strings.Builder

	start := s.GetRange().GetStart()
	end := s.GetRange().GetEnd()
	if start.GetColumn() > 0 || end.GetColumn() > 0 {
		return buildNonLineBasedSuggestion(c, s)
	}

	// we might need to use 4 or more backticks
	//
	// https://docs.gitlab.com/ee/user/project/merge_requests/reviews/suggestions.html#code-block-nested-in-suggestions
	// > If you need to make a suggestion that involves a fenced code block, wrap your suggestion in four backticks instead of the usual three.
	//
	// The documentation doesn't explicitly say anything about cases more than 4 backticks,
	// however it seems to be handled as intended.
	txt := s.GetText()
	backticks := commentutil.GetCodeFenceLength(txt)

	lines := strconv.Itoa(int(s.Range.End.Line - s.Range.Start.Line))
	sb.Grow(backticks + len("suggestion:-0+\n") + len(lines) + len(txt) + len("\n") + backticks)
	commentutil.WriteCodeFence(&sb, backticks)
	sb.WriteString("suggestion:-0+")
	sb.WriteString(lines)
	sb.WriteString("\n")
	if txt != "" {
		sb.WriteString(txt)
		sb.WriteString("\n")
	}
	commentutil.WriteCodeFence(&sb, backticks)

	return sb.String(), nil
}

func buildNonLineBasedSuggestion(c *reviewdog.Comment, s *rdf.Suggestion) (string, error) {
	sourceLines := c.Result.SourceLines
	if len(sourceLines) == 0 {
		return "", errors.New("source lines are not available")
	}
	start := s.GetRange().GetStart()
	end := s.GetRange().GetEnd()
	startLineContent, err := getSourceLine(sourceLines, int(start.GetLine()))
	if err != nil {
		return "", err
	}
	endLineContent, err := getSourceLine(sourceLines, int(end.GetLine()))
	if err != nil {
		return "", err
	}

	txt := startLineContent[:max(start.GetColumn()-1, 0)] + s.GetText() + endLineContent[max(end.GetColumn()-1, 0):]
	backticks := commentutil.GetCodeFenceLength(txt)
	lines := strconv.Itoa(int(end.GetLine() - start.GetLine()))

	var sb strings.Builder
	sb.Grow(backticks + len("suggestion:-0+\n") + len(lines) + len(txt) + len("\n") + backticks)
	commentutil.WriteCodeFence(&sb, backticks)
	sb.WriteString("suggestion:-0+")
	sb.WriteString(lines)
	sb.WriteString("\n")
	if txt != "" {
		sb.WriteString(txt)
		sb.WriteString("\n")
	}
	commentutil.WriteCodeFence(&sb, backticks)
	return sb.String(), nil
}

func getSourceLine(sourceLines map[int]string, line int) (string, error) {
	lineContent, ok := sourceLines[line]
	if !ok {
		return "", fmt.Errorf("source line (L=%d) is not available for this suggestion", line)
	}
	return lineContent, nil
}
