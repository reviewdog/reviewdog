package main

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/doghouse"
	"github.com/haya14busa/reviewdog/doghouse/client"
	"github.com/haya14busa/reviewdog/project"
	"golang.org/x/oauth2"
)

func TestNewDoghouseCli(t *testing.T) {
	if _, ok := newDoghouseCli(context.Background()).Client.Transport.(*oauth2.Transport); ok {
		t.Error("got oauth2 http client, want default client")
	}

	const tokenEnv = "REVIEWDOG_TOKEN"
	saveToken := os.Getenv(tokenEnv)
	defer func() {
		if saveToken != "" {
			os.Setenv(tokenEnv, saveToken)
		} else {
			os.Unsetenv(tokenEnv)
		}
	}()
	os.Setenv(tokenEnv, "xxx")

	if _, ok := newDoghouseCli(context.Background()).Client.Transport.(*oauth2.Transport); !ok {
		t.Error("w/ TOKEN: got unexpected http client, want oauth client")
	}
}

func TestCheckResultSet_Project(t *testing.T) {
	defer func(f func(ctx context.Context, conf *project.Config) (map[string][]*reviewdog.CheckResult, error)) {
		projectRunAndParse = f
	}(projectRunAndParse)

	wantCheckResult := map[string][]*reviewdog.CheckResult{
		"name1": {
			&reviewdog.CheckResult{
				Lnum:    1,
				Col:     14,
				Message: "msg",
				Path:    "reviewdog.go",
			},
		},
	}

	projectRunAndParse = func(ctx context.Context, conf *project.Config) (map[string][]*reviewdog.CheckResult, error) {
		return wantCheckResult, nil
	}

	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	got, err := checkResultSet(context.Background(), nil, &option{conf: tmp.Name()}, true)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, wantCheckResult); diff != "" {
		t.Errorf("result has diff:\n%s", diff)
	}
}

func TestCheckResultSet_NonProject(t *testing.T) {
	opt := &option{
		f: "golint",
	}
	input := `reviewdog.go:14:14: test message`
	got, err := checkResultSet(context.Background(), strings.NewReader(input), opt, false)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]*reviewdog.CheckResult{
		"golint": {
			&reviewdog.CheckResult{
				Lnum:    14,
				Col:     14,
				Message: "test message",
				Path:    "reviewdog.go",
				Lines:   []string{input},
			},
		},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("result has diff:\n%s", diff)
	}
}

type fakeDoghouseCli struct {
	client.DogHouseClientInterface
	FakeCheck func(context.Context, *doghouse.CheckRequest) (*doghouse.CheckResponse, error)
}

func (f *fakeDoghouseCli) Check(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
	return f.FakeCheck(ctx, req)
}

func TestPostResultSet(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	fakeCli := &fakeDoghouseCli{}
	fakeCli.FakeCheck = func(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
		if req.Owner != owner {
			t.Errorf("req.Owner = %q, want %q", req.Owner, owner)
		}
		if req.Repo != repo {
			t.Errorf("req.Repo = %q, want %q", req.Repo, repo)
		}
		if req.SHA != sha {
			t.Errorf("req.SHA = %q, want %q", req.SHA, sha)
		}
		if req.PullRequest != prNum {
			t.Errorf("req.PullRequest = %d, want %d", req.PullRequest, prNum)
		}
		switch req.Name {
		case "name1":
			if diff := cmp.Diff(req.Annotations, []*doghouse.Annotation{
				{
					Line:       14,
					Message:    "name1: test 1",
					Path:       "reviewdog.go",
					RawMessage: "L1\nL2",
				},
				{
					Message: "name1: test 2",
					Path:    "reviewdog.go",
				},
			}); diff != "" {
				t.Errorf("%s: req.Annotation have diff:\n%s", req.Name, diff)
			}
		case "name2":
			if diff := cmp.Diff(req.Annotations, []*doghouse.Annotation{
				{
					Line:    14,
					Message: "name2: test 1",
					Path:    "cmd/reviewdog/doghouse.go",
				},
			}); diff != "" {
				t.Errorf("%s: req.Annotation have diff:\n%s", req.Name, diff)
			}
		default:
			t.Errorf("unexpected req.Name: %s", req.Name)
		}
		return &doghouse.CheckResponse{}, nil
	}

	resultSet := map[string][]*reviewdog.CheckResult{
		"name1": {
			&reviewdog.CheckResult{
				Lnum:    14,
				Message: "name1: test 1",
				Path:    "reviewdog.go",
				Lines:   []string{"L1", "L2"},
			},
			&reviewdog.CheckResult{
				Message: "name1: test 2",
				Path:    "reviewdog.go",
			},
		},
		"name2": {
			&reviewdog.CheckResult{
				Lnum:    14,
				Message: "name2: test 1",
				Path:    "cmd/reviewdog/doghouse.go",
			},
		},
	}

	ghInfo := &PullRequestInfo{
		owner: owner,
		repo:  repo,
		pr:    prNum,
		sha:   sha,
	}

	if err := postResultSet(context.Background(), resultSet, ghInfo, fakeCli); err != nil {
		t.Fatal(err)
	}
}
