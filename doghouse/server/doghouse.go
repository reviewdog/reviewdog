package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v60/github"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
	ghService "github.com/reviewdog/reviewdog/service/github"
)

// GitHub check runs API cannot handle too large requests.
// Set max number of filtered findings to be shown in check-run summary.
// ERROR:
//
//	https://api.github.com/repos/easymotion/vim-easymotion/check-runs: 422
//	Invalid request.
//	Only 65535 characters are allowed; 250684 were supplied. []
const maxAllowedSize = 65535

// > The Checks API limits the number of annotations to a maximum of 50 per API
// > request.
// https://developer.github.com/v3/checks/runs/#output-object
const maxAnnotationsPerRequest = 50

type Checker struct {
	req              *doghouse.CheckRequest
	gh               checkerGitHubClientInterface
	cli              *github.Client
	inDogHouseServer bool // If true, this checker runs in the DogHouse server.
}

func NewChecker(req *doghouse.CheckRequest, gh *github.Client, inDogHouseServer bool) *Checker {
	return &Checker{
		req:              req,
		gh:               &checkerGitHubClient{Client: gh},
		cli:              gh,
		inDogHouseServer: inDogHouseServer,
	}
}

func (ch *Checker) Check(ctx context.Context) (*doghouse.CheckResponse, error) {
	// Get Diff
	var filediffs []*diff.FileDiff
	if ch.req.PullRequest != 0 {
		var err error
		filediffs, err = ch.pullRequestDiff(ctx, ch.req.PullRequest)
		if err != nil {
			return nil, fmt.Errorf("fail to parse diff: %w", err)
		}
	}

	// Convert annotations
	results := annotationsToDiagnostics(ch.req.Annotations)

	// Filter results
	filterMode := ch.req.FilterMode
	//lint:ignore SA1019 Need to support OutsideDiff for backward compatibility.
	if ch.req.PullRequest == 0 || ch.req.OutsideDiff {
		// If it's not Pull Request run, do not filter results by diff regardless
		// of the filter mode.
		filterMode = filter.ModeNoFilter
	}
	filtered := filter.FilterCheck(results, filediffs, 1, "", filterMode)

	// Post annotations
	checkService := &ghService.Check{
		CLI:      ch.cli,
		Owner:    ch.req.Owner,
		Repo:     ch.req.Repo,
		PR:       ch.req.PullRequest,
		SHA:      ch.req.SHA,
		ToolName: ch.req.Name,
		Level:    ch.req.Level,
	}
	for _, f := range filtered {
		if err := checkService.Post(ctx, &reviewdog.Comment{
			Result:   f,
			ToolName: ch.req.Name,
		}); err != nil {
			return nil, err
		}
	}
	if err := checkService.Flush(ctx); err != nil {
		return nil, err
	}
	result := checkService.GetResult()
	if result == nil {
		return nil, errors.New("empty check service result")
	}
	return &doghouse.CheckResponse{
		ReportURL:  result.ReportURL,
		Conclusion: result.Conclusion,
	}, nil
}

func (ch *Checker) pullRequestDiff(ctx context.Context, pr int) ([]*diff.FileDiff, error) {
	d, err := ch.rawPullRequestDiff(ctx, pr)
	if err != nil {
		return nil, err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %w", err)
	}
	return filediffs, nil
}

func (ch *Checker) rawPullRequestDiff(ctx context.Context, pr int) ([]byte, error) {
	return (&ghService.PullRequestDiffService{
		Cli:              ch.cli,
		Owner:            ch.req.Owner,
		Repo:             ch.req.Repo,
		PR:               pr,
		SHA:              ch.req.SHA,
		FallBackToGitCLI: !ch.inDogHouseServer,
	}).Diff(ctx)
}

func annotationsToDiagnostics(as []*doghouse.Annotation) []*rdf.Diagnostic {
	ds := make([]*rdf.Diagnostic, 0, len(as))
	for _, a := range as {
		ds = append(ds, annotationToDiagnostic(a))
	}
	return ds
}

func annotationToDiagnostic(a *doghouse.Annotation) *rdf.Diagnostic {
	if a.Diagnostic != nil {
		return a.Diagnostic
	}
	// Old reviewdog CLI doesn't have the Diagnostic field.
	return &rdf.Diagnostic{
		Location: &rdf.Location{
			//lint:ignore SA1019 use deprecated fields because of backward compatibility.
			Path: a.Path,
			Range: &rdf.Range{
				Start: &rdf.Position{
					//lint:ignore SA1019 use deprecated fields because of backward compatibility.
					Line: int32(a.Line),
				},
			},
		},
		//lint:ignore SA1019 use deprecated fields because of backward compatibility.
		Message: a.Message,
		//lint:ignore SA1019 use deprecated fields because of backward compatibility.
		OriginalOutput: a.RawMessage,
	}
}
