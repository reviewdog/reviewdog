package gitlab

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
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
	toolName string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	postedcs commentutil.PostedComments
	// outdatedDiscussions holds resolvable discussions previously posted by
	// reviewdog that are candidates for auto-resolve if no longer reported.
	// Keyed by fingerprint; value is a slice so fingerprint collisions across
	// discussions do not silently drop entries.
	outdatedDiscussions map[string][]string // fingerprint -> []discussionID
}

// NewGitLabMergeRequestDiscussionCommenter returns a new MergeRequestDiscussionCommenter service.
// MergeRequestDiscussionCommenter service needs git command in $PATH.
func NewGitLabMergeRequestDiscussionCommenter(cli *gitlab.Client, owner, repo string, pr int, sha, toolName string) *MergeRequestDiscussionCommenter {
	return &MergeRequestDiscussionCommenter{
		cli:      cli,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
		toolName: toolName,
	}
}

// Post accepts a comment and holds it. Flush method actually posts comments to
// GitLab in parallel.
func (g *MergeRequestDiscussionCommenter) Post(_ context.Context, c *reviewdog.Comment) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

func (*MergeRequestDiscussionCommenter) ShouldPrependGitRelDir() bool { return true }

// Flush posts comments which has not been posted yet.
func (g *MergeRequestDiscussionCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	defer func() { g.postComments = nil }()
	if err := g.setPostedComments(); err != nil {
		return fmt.Errorf("failed to create posted comments: %w", err)
	}
	if err := g.postCommentsForEach(ctx); err != nil {
		return err
	}
	return g.resolveOutdatedDiscussions(ctx)
}

// setPostedComments lists existing merge request discussions and records the
// ones previously posted by reviewdog (identified by the embedded meta
// comment). Resolvable, unresolved discussions authored by this tool are
// tracked as potentially outdated and will be resolved by
// resolveOutdatedDiscussions unless the diagnostic is reported again in this
// run.
func (g *MergeRequestDiscussionCommenter) setPostedComments() error {
	g.postedcs = make(commentutil.PostedComments)
	g.outdatedDiscussions = make(map[string][]string)
	discussions, err := listAllMergeRequestDiscussion(g.cli, g.projects, g.pr, &gitlab.ListMergeRequestDiscussionsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list all merge request discussions: %w", err)
	}
	for _, d := range discussions {
		for _, note := range d.Notes {
			pos := note.Position
			if pos == nil || pos.NewPath == "" || pos.NewLine == 0 || note.Body == "" {
				continue
			}
			if meta := serviceutil.ExtractMetaComment(note.Body); meta != nil {
				g.postedcs.AddPostedComment(pos.NewPath, int(pos.NewLine), meta.GetFingerprint())
				// Only track notes authored by this tool. A non-empty toolName
				// is required so that meta comments with an unset SourceName
				// are never auto-resolved across unrelated tools.
				if g.toolName != "" && meta.GetSourceName() == g.toolName && note.Resolvable && !note.Resolved {
					fp := meta.GetFingerprint()
					g.outdatedDiscussions[fp] = append(g.outdatedDiscussions[fp], d.ID)
				}
				continue
			}
			// Back-compat: notes posted before meta comments were added are
			// matched by raw body text to avoid duplicate posts. These legacy
			// discussions cannot be auto-resolved (no fingerprint) and will
			// need to be resolved manually.
			g.postedcs.AddPostedComment(pos.NewPath, int(pos.NewLine), note.Body)
		}
	}
	return nil
}

func (g *MergeRequestDiscussionCommenter) postCommentsForEach(ctx context.Context) error {
	mr, _, err := g.cli.MergeRequests.GetMergeRequest(g.projects, int64(g.pr), nil, gitlab.WithContext(ctx))
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
		if !c.Result.InDiffFile || lnum == 0 {
			continue
		}
		fprint, err := serviceutil.Fingerprint(c.Result.Diagnostic)
		if err != nil {
			return err
		}
		if g.postedcs.IsPosted(c, lnum, fprint) {
			// Still reported — not outdated.
			delete(g.outdatedDiscussions, fprint)
			continue
		}
		legacyBody := commentutil.MarkdownComment(c)
		if suggestion := buildSuggestions(c); suggestion != "" {
			legacyBody = legacyBody + "\n\n" + suggestion
		}
		// Back-compat: notes posted before meta comments were introduced are
		// indexed by raw body text in postedcs. Skip re-posting if the legacy
		// body matches. Legacy notes cannot be auto-resolved.
		if g.postedcs.IsPosted(c, lnum, legacyBody) {
			continue
		}
		body := legacyBody + fmt.Sprintf("\n%s\n", serviceutil.BuildMetaComment(fprint, g.toolName))
		eg.Go(func() error {
			pos := &gitlab.PositionOptions{
				StartSHA:     gitlab.Ptr(targetBranch.Commit.ID),
				HeadSHA:      gitlab.Ptr(g.sha),
				BaseSHA:      gitlab.Ptr(targetBranch.Commit.ID),
				PositionType: gitlab.Ptr("text"),
				NewPath:      gitlab.Ptr(loc.GetPath()),
				NewLine:      gitlab.Ptr(int64(lnum)),
			}
			if c.Result.OldPath != "" && c.Result.OldLine != 0 {
				pos.OldPath = gitlab.Ptr(c.Result.OldPath)
				pos.OldLine = gitlab.Ptr(int64(c.Result.OldLine))
			}
			discussion := &gitlab.CreateMergeRequestDiscussionOptions{
				Body:     gitlab.Ptr(body),
				Position: pos,
			}
			_, _, err := g.cli.Discussions.CreateMergeRequestDiscussion(g.projects, int64(g.pr), discussion)
			if err != nil {
				return fmt.Errorf("failed to create merge request discussion: %w", err)
			}
			return nil
		})
	}
	return eg.Wait()
}

// resolveOutdatedDiscussions marks previously-posted reviewdog discussions as
// resolved when the corresponding diagnostic is no longer reported in the
// current run. This keeps a MR review clean as fixes land.
//
// All resolve attempts run concurrently; every failure is surfaced via
// errors.Join so operators see the full picture instead of only the first
// error.
func (g *MergeRequestDiscussionCommenter) resolveOutdatedDiscussions(ctx context.Context) error {
	if g.toolName == "" || len(g.outdatedDiscussions) == 0 {
		return nil
	}
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)
	for _, ids := range g.outdatedDiscussions {
		for _, id := range ids {
			wg.Add(1)
			go func(discussionID string) {
				defer wg.Done()
				_, _, err := g.cli.Discussions.ResolveMergeRequestDiscussion(
					g.projects, int64(g.pr), discussionID,
					&gitlab.ResolveMergeRequestDiscussionOptions{Resolved: gitlab.Ptr(true)},
					gitlab.WithContext(ctx),
				)
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("failed to resolve merge request discussion (id=%s): %w", discussionID, err))
					mu.Unlock()
				}
			}(id)
		}
	}
	wg.Wait()
	return errors.Join(errs...)
}

func listAllMergeRequestDiscussion(cli *gitlab.Client, projectID string, mergeRequest int, opts *gitlab.ListMergeRequestDiscussionsOptions) ([]*gitlab.Discussion, error) {
	discussions, resp, err := cli.Discussions.ListMergeRequestDiscussions(projectID, int64(mergeRequest), opts)
	if err != nil {
		return nil, err
	}
	if resp.NextPage == 0 {
		return discussions, nil
	}
	newOpts := &gitlab.ListMergeRequestDiscussionsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    resp.NextPage,
			PerPage: opts.PerPage,
		},
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
