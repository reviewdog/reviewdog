package server

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v26/github"
	"github.com/reviewdog/reviewdog/doghouse"
)

type fakeCheckerGitHubCli struct {
	checkerGitHubClientInterface
	FakeGetPullRequest     func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error)
	FakeGetPullRequestDiff func(ctx context.Context, owner, repo string, number int) ([]byte, error)
	FakeCreateCheckRun     func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error)
}

func (f *fakeCheckerGitHubCli) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	return f.FakeGetPullRequest(ctx, owner, repo, number)
}

func (f *fakeCheckerGitHubCli) GetPullRequestDiff(ctx context.Context, owner, repo string, number int) ([]byte, error) {
	return f.FakeGetPullRequestDiff(ctx, owner, repo, number)
}

func (f *fakeCheckerGitHubCli) CreateCheckRun(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
	return f.FakeCreateCheckRun(ctx, owner, repo, opt)
}

const sampleDiff = `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
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
		name      = "haya14busa-linter"
		owner     = "haya14busa"
		repo      = "reviewdog"
		prNum     = 14
		sha       = "1414"
		reportURL = "http://example.com/report_url"
		branch    = "test-branch"
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Annotations: []*doghouse.Annotation{
			{
				Path:       "sample.new.txt",
				Line:       2,
				Message:    "test message",
				RawMessage: "raw test message",
			},
			{
				Path:       "sample.new.txt",
				Line:       14,
				Message:    "test message outside diff",
				RawMessage: "raw test message outside diff",
			},
		},
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		if number != prNum {
			t.Errorf("PullRequest number = %d, want %d", number, prNum)
		}
		return &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String(branch),
			},
		}, nil
	}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		if opt.Name != name {
			t.Errorf("CreateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		if opt.HeadBranch != branch {
			t.Errorf("CreateCheckRunOptions.HeadBranch = %q, want %q", opt.HeadBranch, branch)
		}
		if opt.HeadSHA != sha {
			t.Errorf("CreateCheckRunOptions.HeadSHA = %q, want %q", opt.HeadSHA, sha)
		}
		annotations := opt.Output.Annotations
		wantAnnotaions := []*github.CheckRunAnnotation{
			{
				Path:            github.String("sample.new.txt"),
				BlobHRef:        github.String("http://github.com/haya14busa/reviewdog/blob/1414/sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("test message"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
				RawDetails:      github.String("raw test message"),
			},
		}
		if d := cmp.Diff(annotations, wantAnnotaions); d != "" {
			t.Errorf("Annotation diff found:\n%s", d)
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

func TestCheck_branch_in_req(t *testing.T) {
	const (
		name      = "haya14busa-linter"
		owner     = "haya14busa"
		repo      = "reviewdog"
		prNum     = 14
		sha       = "1414"
		reportURL = "http://example.com/report_url"
		branch    = "test-branch"
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Branch:      branch,
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		t.Fatal("GetPullRequest should not be called")
		return nil, nil
	}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		if opt.HeadBranch != branch {
			t.Errorf("CreateCheckRunOptions.HeadBranch = %q, want %q", opt.HeadBranch, branch)
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	if _, err := checker.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_fail_get_pullrequest(t *testing.T) {
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		return nil, errors.New("test failrue")
	}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
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

func TestCheck_fail_empty_branch(t *testing.T) {
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		return &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String(""),
			},
		}, nil
	}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
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

func TestCheck_fail_diff(t *testing.T) {
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		return &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String("branch"),
			},
		}, nil
	}
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
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		return &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String("branch"),
			},
		}, nil
	}
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
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequest = func(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
		return &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String("branch"),
			},
		}, nil
	}
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
