package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

func TestGitHubAnnotationsWithoutToken(t *testing.T) {
	t.Setenv("REVIEWDOG_GITHUB_API_TOKEN", "")
	for _, reporter := range []string{"github-annotations", "github-pr-annotations"} {
		t.Run(reporter, func(t *testing.T) {
			opt := &option{
				reporter:   reporter,
				efms:       strslice([]string{"%f:%l: %m"}),
				filterMode: filter.ModeNoFilter,
			}
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			old := os.Stdout
			os.Stdout = write
			err = run(strings.NewReader("file.go:1: example issue\n"), new(bytes.Buffer), opt)
			os.Stdout = old
			write.Close()
			output, readErr := io.ReadAll(read)
			read.Close()
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !strings.Contains(string(output), "example issue") || !strings.Contains(string(output), "::error") {
				t.Fatalf("missing annotation: %q", output)
			}
		})
	}
}

func TestGitHubAnnotationsFilteredStillRequiresToken(t *testing.T) {
	t.Setenv("REVIEWDOG_GITHUB_API_TOKEN", "")
	for _, reporter := range []string{"github-annotations", "github-pr-annotations"} {
		for _, mode := range []filter.Mode{filter.ModeDefault, filter.ModeAdded} {
			opt := &option{reporter: reporter, efms: strslice([]string{"%f:%l: %m"}), filterMode: mode}
			err := run(strings.NewReader(""), new(bytes.Buffer), opt)
			if err == nil || !strings.Contains(err.Error(), "REVIEWDOG_GITHUB_API_TOKEN") {
				t.Errorf("reporter %s mode %v: got %v, want missing token error", reporter, mode, err)
			}
		}
	}
}

// A PR diff is fetched only for filtered reports. Both reporters must still
// apply that diff to the added and unchanged lines.
func TestGitHubAnnotationsPRFiltering(t *testing.T) {
	var requests atomic.Int32
	rel, err := serviceutil.GitRelWorkdir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.ToSlash(filepath.Join(rel, "example.go"))
	diff := "diff --git a/" + path + " b/" + path + "\n--- a/" + path + "\n+++ b/" + path + "\n@@ -1,2 +1,2 @@\n-old\n+changed\n unchanged\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/repos/example/repo/pulls/123" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, diff)
	}))
	defer server.Close()
	for name, value := range map[string]string{
		"GITHUB_ACTIONS": "", "TRAVIS_REPO_SLUG": "", "DRONE_REPO": "", "BITBUCKET_REPO_FULL_NAME": "",
		"CI_REPO_OWNER": "example", "CI_REPO_NAME": "repo", "CI_COMMIT": "abc", "CI_PULL_REQUEST": "123",
		"REVIEWDOG_GITHUB_API_TOKEN": "dummy-token", "GITHUB_API": server.URL + "/", "GITHUB_API_URL": "",
	} {
		t.Setenv(name, value)
	}
	for _, reporter := range []string{"github-annotations", "github-pr-annotations"} {
		for _, mode := range []filter.Mode{filter.ModeDefault, filter.ModeAdded, filter.ModeNoFilter} {
			requests.Store(0)
			options := &option{reporter: reporter, efms: strslice([]string{"%f:%l: %m"}), filterMode: mode, logLevel: "error"}
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			old := os.Stdout
			os.Stdout = write
			err = run(strings.NewReader("example.go:1: added-result\nexample.go:2: context-result\n"), new(bytes.Buffer), options)
			os.Stdout = old
			write.Close()
			output, readErr := io.ReadAll(read)
			read.Close()
			if err != nil {
				t.Fatalf("%s %v: %v", reporter, mode, err)
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			text := string(output)
			if !strings.Contains(text, "added-result") {
				t.Errorf("%s %v: missing added-result: %q", reporter, mode, text)
			}
			if got := strings.Contains(text, "context-result"); got != (mode == filter.ModeNoFilter) {
				t.Errorf("%s %v: unexpected context-result=%v: %q", reporter, mode, got, text)
			}
			wantRequests := int32(1)
			if mode == filter.ModeNoFilter {
				wantRequests = 0
			}
			if got := requests.Load(); got != wantRequests {
				t.Errorf("%s %v: requests=%d want=%d", reporter, mode, got, wantRequests)
			}
		}
	}
	// Without a PR number, -guess normally searches the API. In nofilter mode
	// the lookup is unnecessary, even when guessing is requested.
	t.Setenv("CI_PULL_REQUEST", "")
	for _, name := range []string{"TRAVIS_PULL_REQUEST", "CIRCLE_PULL_REQUEST", "CIRCLE_PR_NUMBER", "DRONE_PULL_REQUEST", "CI_MERGE_REQUEST_IID", "BITBUCKET_PR_ID", "CI_COMMIT_PULL_REQUEST"} {
		t.Setenv(name, "")
	}
	oldGuess := opt.guessPullRequest
	opt.guessPullRequest = true
	defer func() { opt.guessPullRequest = oldGuess }()
	for _, reporter := range []string{"github-annotations", "github-pr-annotations"} {
		requests.Store(0)
		options := &option{reporter: reporter, efms: strslice([]string{"%f:%l: %m"}), filterMode: filter.ModeNoFilter, guessPullRequest: true, logLevel: "error"}
		read, write, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		old := os.Stdout
		os.Stdout = write
		err = run(strings.NewReader("example.go:1: guessed finding\n"), new(bytes.Buffer), options)
		os.Stdout = old
		write.Close()
		output, readErr := io.ReadAll(read)
		read.Close()
		if err != nil {
			t.Fatalf("%s with guess: %v", reporter, err)
		}
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(output), "guessed finding") || requests.Load() != 0 {
			t.Errorf("%s with guess: output=%q requests=%d", reporter, output, requests.Load())
		}
	}

}
