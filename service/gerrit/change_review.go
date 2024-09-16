package gerrit

import (
	"context"
	"sync"

	"golang.org/x/build/gerrit"

	"github.com/reviewdog/reviewdog"
)

var _ reviewdog.CommentService = &ChangeReviewCommenter{}

// ChangeReviewCommenter is a comment service for Gerrit Change Review
// API:
//
//	https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#set-review
//	POST /changes/{change-id}/revisions/{revision-id}/review
type ChangeReviewCommenter struct {
	cli        *gerrit.Client
	changeID   string
	revisionID string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment
}

// NewChangeReviewCommenter returns a new NewChangeReviewCommenter service.
// ChangeReviewCommenter service needs git command in $PATH.
func NewChangeReviewCommenter(cli *gerrit.Client, changeID, revisionID string) *ChangeReviewCommenter {
	return &ChangeReviewCommenter{
		cli:          cli,
		changeID:     changeID,
		revisionID:   revisionID,
		postComments: []*reviewdog.Comment{},
	}
}

// Post accepts a comment and holds it. Flush method actually posts comments to Gerrit
func (g *ChangeReviewCommenter) Post(_ context.Context, c *reviewdog.Comment) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

func (*ChangeReviewCommenter) ShouldPrependGitRelDir() bool { return true }

// Flush posts comments which has not been posted yet.
func (g *ChangeReviewCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()
	defer func() { g.postComments = nil }()

	return g.postAllComments(ctx)
}

func (g *ChangeReviewCommenter) postAllComments(ctx context.Context) error {
	review := gerrit.ReviewInput{
		Comments: map[string][]gerrit.CommentInput{},
	}
	for _, c := range g.postComments {
		if !c.Result.InDiffFile {
			continue
		}
		loc := c.Result.Diagnostic.GetLocation()
		path := loc.GetPath()
		review.Comments[path] = append(review.Comments[path], gerrit.CommentInput{
			Line:    int(loc.GetRange().GetStart().GetLine()),
			Message: c.Result.Diagnostic.GetMessage(),
		})
	}

	return g.cli.SetReview(ctx, g.changeID, g.revisionID, review)
}
