package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/doghouse/client"
	"github.com/reviewdog/reviewdog/project"
	"golang.org/x/oauth2"
)

func setupEnvs(testEnvs map[string]string) (cleanup func()) {
	saveEnvs := make(map[string]string)
	for key, value := range testEnvs {
		saveEnvs[key] = os.Getenv(key)
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
	return func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}
}

func TestNewDoghouseCli_returnGitHubClient(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "",
		"GITHUB_ACTION":              "xxx",
		"REVIEWDOG_GITHUB_API_TOKEN": "xxx",
	})
	defer cleanup()
	cli, err := newDoghouseCli(context.Background())
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if _, ok := cli.(*client.GitHubClient); !ok {
		t.Errorf("got %T client, want *client.GitHubClient client", cli)
	}
}

func TestNewDoghouseCli_returnErrorForGitHubClient(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "",
		"GITHUB_ACTION":              "xxx",
		"REVIEWDOG_GITHUB_API_TOKEN": "", // missing
	})
	defer cleanup()
	if _, err := newDoghouseCli(context.Background()); err == nil {
		t.Error("got no error but want REVIEWDOG_GITHUB_API_TOKEN missing error")
	}
}

func TestNewDoghouseCli_returnDogHouseClientWithReviewdogToken(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "xxx",
		"GITHUB_ACTION":              "xxx",
		"REVIEWDOG_GITHUB_API_TOKEN": "xxx",
	})
	defer cleanup()
	cli, err := newDoghouseCli(context.Background())
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if _, ok := cli.(*client.DogHouseClient); !ok {
		t.Errorf("got %T client, want *client.DogHouseClient client", cli)
	}
}

func TestNewDoghouseCli_returnDogHouseClient(t *testing.T) {
	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN":            "",
		"GITHUB_ACTION":              "",
		"REVIEWDOG_GITHUB_API_TOKEN": "",
	})
	defer cleanup()
	cli, err := newDoghouseCli(context.Background())
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if _, ok := cli.(*client.DogHouseClient); !ok {
		t.Errorf("got %T client, want *client.DogHouseClient client", cli)
	}
}

func TestNewDoghouseServerCli(t *testing.T) {
	if _, ok := newDoghouseServerCli(context.Background()).Client.Transport.(*oauth2.Transport); ok {
		t.Error("got oauth2 http client, want default client")
	}

	cleanup := setupEnvs(map[string]string{
		"REVIEWDOG_TOKEN": "xxx",
	})
	defer cleanup()

	if _, ok := newDoghouseServerCli(context.Background()).Client.Transport.(*oauth2.Transport); !ok {
		t.Error("w/ TOKEN: got unexpected http client, want oauth client")
	}
}

func TestCheckResultSet_Project(t *testing.T) {
	defer func(f func(ctx context.Context, conf *project.Config) (*reviewdog.ResultMap, error)) {
		projectRunAndParse = f
	}(projectRunAndParse)

	var wantCheckResult reviewdog.ResultMap
	wantCheckResult.Store("name1", []*reviewdog.CheckResult{
		{
			Lnum:    1,
			Col:     14,
			Message: "msg",
			Path:    "reviewdog.go",
		},
	})

	projectRunAndParse = func(ctx context.Context, conf *project.Config) (*reviewdog.ResultMap, error) {
		return &wantCheckResult, nil
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

	if got.Len() != wantCheckResult.Len() {
		t.Errorf("length of results is different. got = %d, want = %d\n", got.Len(), wantCheckResult.Len())
	}
	got.Range(func(k string, v []*reviewdog.CheckResult) {
		w, _ := wantCheckResult.Load(k)
		if diff := cmp.Diff(v, w); diff != "" {
			t.Errorf("result has diff:\n%s", diff)
		}
	})
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
	var want reviewdog.ResultMap
	want.Store("golint", []*reviewdog.CheckResult{
		{
			Lnum:    14,
			Col:     14,
			Message: "test message",
			Path:    "reviewdog.go",
			Lines:   []string{input},
		},
	})

	if got.Len() != want.Len() {
		t.Errorf("length of results is different. got = %d, want = %d\n", got.Len(), want.Len())
	}
	got.Range(func(k string, v []*reviewdog.CheckResult) {
		w, _ := want.Load(k)
		if diff := cmp.Diff(v, w); diff != "" {
			t.Errorf("result has diff:\n%s", diff)
		}
	})
}

type fakeDoghouseServerCli struct {
	client.DogHouseClientInterface
	FakeCheck func(context.Context, *doghouse.CheckRequest) (*doghouse.CheckResponse, error)
}

func (f *fakeDoghouseServerCli) Check(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
	return f.FakeCheck(ctx, req)
}

func TestPostResultSet(t *testing.T) {
	const (
		owner = "haya14busa"
		repo  = "reviewdog"
		prNum = 14
		sha   = "1414"
	)

	fakeCli := &fakeDoghouseServerCli{}
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

	var resultSet reviewdog.ResultMap
	resultSet.Store("name1", []*reviewdog.CheckResult{
		{
			Lnum:    14,
			Message: "name1: test 1",
			Path:    "reviewdog.go",
			Lines:   []string{"L1", "L2"},
		},
		{
			Message: "name1: test 2",
			Path:    "reviewdog.go",
		},
	})
	resultSet.Store("name2", []*reviewdog.CheckResult{
		{
			Lnum:    14,
			Message: "name2: test 1",
			Path:    "cmd/reviewdog/doghouse.go",
		},
	})

	ghInfo := &cienv.BuildInfo{
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
	}

	if _, err := postResultSet(context.Background(), &resultSet, ghInfo, fakeCli); err != nil {
		t.Fatal(err)
	}
}

func TestReportResults(t *testing.T) {
	filteredResultSet := new(reviewdog.FilteredCheckMap)
	filteredResultSet.Store("name1", []*reviewdog.FilteredCheck{
		{
			CheckResult: &reviewdog.CheckResult{
				Lines: []string{"name1-L1", "name1-L2"},
			},
			InDiff: true,
		},
		{
			CheckResult: &reviewdog.CheckResult{
				Lines: []string{"name1.2-L1", "name1.2-L2"},
			},
			InDiff: false,
		},
	})
	filteredResultSet.Store("name2", []*reviewdog.FilteredCheck{
		{
			CheckResult: &reviewdog.CheckResult{
				Lines: []string{"name1-L1", "name1-L2"},
			},
			InDiff: false,
		},
	})
	stdout := new(bytes.Buffer)
	foundResultInDiff := reportResults(stdout, filteredResultSet)
	if !foundResultInDiff {
		t.Errorf("foundResultInDiff = %v, want true", foundResultInDiff)
	}
	want := `reviwedog: Reporting results for "name1"
name1-L1
name1-L2
reviwedog: Reporting results for "name2"
reviwedog: No results found for "name2"
`
	if got := stdout.String(); got != want {
		t.Errorf("diff found for report:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestReportResults_noResultsInDiff(t *testing.T) {
	filteredResultSet := new(reviewdog.FilteredCheckMap)
	filteredResultSet.Store("name1", []*reviewdog.FilteredCheck{
		{
			CheckResult: &reviewdog.CheckResult{
				Lines: []string{"name1-L1", "name1-L2"},
			},
			InDiff: false,
		},
		{
			CheckResult: &reviewdog.CheckResult{
				Lines: []string{"name1.2-L1", "name1.2-L2"},
			},
			InDiff: false,
		},
	})
	filteredResultSet.Store("name2", []*reviewdog.FilteredCheck{
		{
			CheckResult: &reviewdog.CheckResult{
				Lines: []string{"name1-L1", "name1-L2"},
			},
			InDiff: false,
		},
	})
	stdout := new(bytes.Buffer)
	foundResultInDiff := reportResults(stdout, filteredResultSet)
	if foundResultInDiff {
		t.Errorf("foundResultInDiff = %v, want false", foundResultInDiff)
	}
	want := `reviwedog: Reporting results for "name1"
reviwedog: No results found for "name1"
reviwedog: Reporting results for "name2"
reviwedog: No results found for "name2"
`
	if got := stdout.String(); got != want {
		t.Errorf("diff found for report:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
