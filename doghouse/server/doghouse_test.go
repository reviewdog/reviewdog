package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v38/github"

	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/proto/rdf"
)

type fakeCheckerGitHubCli struct {
	checkerGitHubClientInterface
	FakeGetPullRequestDiff func(ctx context.Context, owner, repo string, number int) ([]byte, error)
	FakeCreateCheckRun     func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error)
	FakeUpdateCheckRun     func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error)
}

func (f *fakeCheckerGitHubCli) GetPullRequestDiff(ctx context.Context, owner, repo string, number int) ([]byte, error) {
	return f.FakeGetPullRequestDiff(ctx, owner, repo, number)
}

func (f *fakeCheckerGitHubCli) CreateCheckRun(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
	return f.FakeCreateCheckRun(ctx, owner, repo, opt)
}

func (f *fakeCheckerGitHubCli) UpdateCheckRun(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
	return f.FakeUpdateCheckRun(ctx, owner, repo, checkID, opt)
}

const sampleDiff = `--- a/sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ b/sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- a/nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ b/nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

func TestCheck_OK(t *testing.T) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		wantCheckID = 1414
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Annotations: []*doghouse.Annotation{
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2, Column: 1},
						},
					},
					OriginalOutput: "raw test message",
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message outside diff",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 14},
						},
					},
					OriginalOutput: "raw test message outside diff",
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test multiline",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
							End:   &rdf.Position{Line: 3},
						},
					},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test multiline with column",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2, Column: 1},
							End:   &rdf.Position{Line: 3, Column: 5},
						},
					},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test range comment",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2, Column: 1},
							End:   &rdf.Position{Line: 2, Column: 5},
						},
					},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message:  "test severity override",
					Severity: rdf.Severity_ERROR,
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "source test",
					Source: &rdf.Source{
						Name: "awesome-linter",
					},
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "code test w/o URL",
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
					Code: &rdf.Code{Value: "CODE14"},
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "code test w/ URL",
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
					Code: &rdf.Code{Value: "CODE14", Url: "https://github.com/reviewdog#CODE14"},
				},
			},
			{
				Path:       "sample.new.txt",
				Line:       2,
				Message:    "request from old clients",
				RawMessage: "raw message from old clients",
			},
		},
		Level: "warning",
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		if opt.Name != name {
			t.Errorf("CreateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		if opt.HeadSHA != sha {
			t.Errorf("CreateCheckRunOptions.HeadSHA = %q, want %q", opt.HeadSHA, sha)
		}
		return &github.CheckRun{ID: github.Int64(wantCheckID)}, nil
	}
	cli.FakeUpdateCheckRun = func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
		if checkID != wantCheckID {
			t.Errorf("UpdateCheckRun: checkID = %d, want %d", checkID, wantCheckID)
		}
		if opt.Name != name {
			t.Errorf("UpdateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		annotations := opt.Output.Annotations
		if len(annotations) == 0 {
			if *opt.Conclusion != conclusion {
				t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
			}
		} else {
			wantAnnotations := []*github.CheckRunAnnotation{
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test message"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
					RawDetails:      github.String("raw test message"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(3),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test multiline"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2-L3"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(3),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test multiline with column"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2-L3"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					StartColumn:     github.Int(1),
					EndColumn:       github.Int(5),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test range comment"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("failure"),
					Message:         github.String("test severity override"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("source test"),
					Title:           github.String("[awesome-linter] sample.new.txt#L2"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("code test w/o URL"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2 <CODE14>"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("code test w/ URL"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2 <CODE14>(https://github.com/reviewdog#CODE14)"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("request from old clients"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
					RawDetails:      github.String("raw message from old clients"),
				},
			}
			if d := cmp.Diff(annotations, wantAnnotations); d != "" {
				t.Errorf("Annotation diff found:\n%s", d)
			}
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	res, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if res.ReportURL != reportURL {
		t.Errorf("res.reportURL = %q, want %q", res.ReportURL, reportURL)
	}
}

func testOutsideDiff(t *testing.T, outsideDiff bool, filterMode filter.Mode) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		wantCheckID = 1414
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Annotations: []*doghouse.Annotation{
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
					OriginalOutput: "raw test message",
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message outside diff",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 14},
						},
					},
					OriginalOutput: "raw test message outside diff",
				},
			},
		},
		Level:       "warning",
		OutsideDiff: outsideDiff,
		FilterMode:  filterMode,
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return &github.CheckRun{ID: github.Int64(wantCheckID)}, nil
	}
	cli.FakeUpdateCheckRun = func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
		annotations := opt.Output.Annotations
		if len(annotations) == 0 {
			if *opt.Conclusion != conclusion {
				t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
			}
		} else {
			wantAnnotations := []*github.CheckRunAnnotation{
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test message"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
					RawDetails:      github.String("raw test message"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(14),
					EndLine:         github.Int(14),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test message outside diff"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L14"),
					RawDetails:      github.String("raw test message outside diff"),
				},
			}
			if d := cmp.Diff(annotations, wantAnnotations); d != "" {
				t.Errorf("Annotation diff found:\n%s", d)
			}
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	if _, err := checker.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_OK_deprecated_outsidediff(t *testing.T) {
	t.Run("deprecated: outside_diff", func(t *testing.T) {
		testOutsideDiff(t, true, filter.ModeDefault)
	})
	t.Run("filter-mode=NoFilter", func(t *testing.T) {
		testOutsideDiff(t, false, filter.ModeNoFilter)
	})
}

func TestCheck_OK_multiple_update_runs(t *testing.T) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		prNum       = 14
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		wantCheckID = 1414
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Level:       "warning",
	}
	for i := 0; i < 101; i++ {
		req.Annotations = append(req.Annotations, &doghouse.Annotation{
			Diagnostic: &rdf.Diagnostic{
				Message: "test message",
				Location: &rdf.Location{
					Path: "sample.new.txt",
					Range: &rdf.Range{
						Start: &rdf.Position{Line: 2},
					},
				},
				OriginalOutput: "raw test message",
			},
		})
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		if opt.Name != name {
			t.Errorf("CreateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		if opt.HeadSHA != sha {
			t.Errorf("CreateCheckRunOptions.HeadSHA = %q, want %q", opt.HeadSHA, sha)
		}
		return &github.CheckRun{ID: github.Int64(wantCheckID)}, nil
	}
	cli.FakeUpdateCheckRun = func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
		if checkID != wantCheckID {
			t.Errorf("UpdateCheckRun: checkID = %d, want %d", checkID, wantCheckID)
		}
		annotations := opt.Output.Annotations
		switch len(annotations) {
		case 0:
			if *opt.Conclusion != conclusion {
				t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
			}
		case maxAnnotationsPerRequest, 1: // Expected
		default:
			t.Errorf("UpdateCheckRun: len(annotations) = %d, but it's unexpected", len(annotations))
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	if _, err := checker.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_OK_nonPullRequests(t *testing.T) {
	const (
		name        = "haya14busa-linter"
		owner       = "haya14busa"
		repo        = "reviewdog"
		sha         = "1414"
		reportURL   = "http://example.com/report_url"
		conclusion  = "neutral"
		wantCheckID = 1414
	)

	req := &doghouse.CheckRequest{
		// Do not set PullRequest
		Name:  name,
		Owner: owner,
		Repo:  repo,
		SHA:   sha,
		Annotations: []*doghouse.Annotation{
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 2},
						},
					},
					OriginalOutput: "raw test message",
				},
			},
			{
				Diagnostic: &rdf.Diagnostic{
					Message: "test message2",
					Location: &rdf.Location{
						Path: "sample.new.txt",
						Range: &rdf.Range{
							Start: &rdf.Position{Line: 14},
						},
					},
					OriginalOutput: "raw test message2",
				},
			},
		},
		Level: "warning",
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		t.Errorf("GetPullRequestDiff should not be called")
		return nil, nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return &github.CheckRun{ID: github.Int64(wantCheckID)}, nil
	}
	cli.FakeUpdateCheckRun = func(ctx context.Context, owner, repo string, checkID int64, opt github.UpdateCheckRunOptions) (*github.CheckRun, error) {
		if checkID != wantCheckID {
			t.Errorf("UpdateCheckRun: checkID = %d, want %d", checkID, wantCheckID)
		}
		annotations := opt.Output.Annotations
		if len(annotations) == 0 {
			if *opt.Conclusion != conclusion {
				t.Errorf("UpdateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
			}
		} else {
			wantAnnotations := []*github.CheckRunAnnotation{
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(2),
					EndLine:         github.Int(2),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test message"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
					RawDetails:      github.String("raw test message"),
				},
				{
					Path:            github.String("sample.new.txt"),
					StartLine:       github.Int(14),
					EndLine:         github.Int(14),
					AnnotationLevel: github.String("warning"),
					Message:         github.String("test message2"),
					Title:           github.String("[haya14busa-linter] sample.new.txt#L14"),
					RawDetails:      github.String("raw test message2"),
				},
			}
			if d := cmp.Diff(annotations, wantAnnotations); d != "" {
				t.Errorf("Annotation diff found:\n%s", d)
			}
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	res, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if res.ReportURL != reportURL {
		t.Errorf("res.reportURL = %q, want %q", res.ReportURL, reportURL)
	}
}

func TestCheck_fail_diff(t *testing.T) {
	req := &doghouse.CheckRequest{PullRequest: 1}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return nil, errors.New("test diff failure")
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return &github.CheckRun{}, nil
	}
	checker := &Checker{req: req, gh: cli}

	if _, err := checker.Check(context.Background()); err == nil {
		t.Fatalf("got no error, want some error")
	} else {
		t.Log(err)
	}
}

func TestCheck_fail_invalid_diff(t *testing.T) {
	t.Skip("Parse invalid diff function somehow doesn't return error")
	req := &doghouse.CheckRequest{PullRequest: 1}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte("invalid diff"), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return &github.CheckRun{}, nil
	}
	checker := &Checker{req: req, gh: cli}

	if _, err := checker.Check(context.Background()); err == nil {
		t.Fatalf("got no error, want some error")
	} else {
		t.Log(err)
	}
}

func TestCheck_fail_check(t *testing.T) {
	req := &doghouse.CheckRequest{PullRequest: 1}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return nil, errors.New("test check failure")
	}
	checker := &Checker{req: req, gh: cli}

	if _, err := checker.Check(context.Background()); err == nil {
		t.Fatalf("got no error, want some error")
	} else {
		t.Log(err)
	}
}

func TestCheck_fail_check_with_403(t *testing.T) {
	req := &doghouse.CheckRequest{PullRequest: 1}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return nil, &github.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
			},
		}
	}
	checker := &Checker{req: req, gh: cli}

	resp, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	if resp.CheckedResults == nil {
		t.Error("resp.CheckedResults should not be nil")
	}
}
