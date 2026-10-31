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

// captureStdout drains the pipe concurrently so large outputs cannot block the writer.
// It must not be used from parallel tests because it replaces the process stdout.
func captureStdout(t *testing.T, fn func() error) (output string, runErr error) {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	type readResult struct {
		data []byte
		err  error
	}
	done := make(chan readResult, 1)
	go func() {
		data, err := io.ReadAll(read)
		done <- readResult{data: data, err: err}
	}()
	old := os.Stdout
	os.Stdout = write
	defer func() {
		os.Stdout = old
		if err := write.Close(); err != nil {
			t.Errorf("close stdout pipe writer: %v", err)
		}
		result := <-done
		if err := read.Close(); err != nil {
			t.Errorf("close stdout pipe reader: %v", err)
		}
		if result.err != nil {
			t.Errorf("read stdout pipe: %v", result.err)
		}
		output = string(result.data)
	}()
	runErr = fn()
	return
}

func TestCaptureStdoutLargeOutput(t *testing.T) {
	want := strings.Repeat("x", 256*1024)
	got, err := captureStdout(t, func() error {
		_, err := io.WriteString(os.Stdout, want)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("captured %d bytes, want %d", len(got), len(want))
	}
}

func TestCaptureStdoutRestoresOnPanic(t *testing.T) {
	old := os.Stdout
	func() {
		defer func() {
			if got := recover(); got != "test panic" {
				t.Errorf("panic = %v, want test panic", got)
			}
		}()
		_, _ = captureStdout(t, func() error { panic("test panic") })
	}()
	if os.Stdout != old {
		t.Error("stdout was not restored after panic")
	}
}

func TestGitHubAnnotationsWithoutToken(t *testing.T) {
	t.Setenv("REVIEWDOG_GITHUB_API_TOKEN", "")
	for _, reporter := range []string{"github-annotations", "github-pr-annotations"} {
		t.Run(reporter, func(t *testing.T) {
			opt := &option{
				reporter:   reporter,
				efms:       strslice([]string{"%f:%l: %m"}),
				filterMode: filter.ModeNoFilter,
			}
			output, err := captureStdout(t, func() error {
				return run(strings.NewReader("file.go:1: example issue\n"), new(bytes.Buffer), opt)
			})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if !strings.Contains(output, "example issue") || !strings.Contains(output, "::error") {
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
			output, err := captureStdout(t, func() error {
				return run(strings.NewReader("example.go:1: added-result\nexample.go:2: context-result\n"), new(bytes.Buffer), options)
			})
			if err != nil {
				t.Fatalf("%s %v: %v", reporter, mode, err)
			}
			text := output
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
		output, err := captureStdout(t, func() error {
			return run(strings.NewReader("example.go:1: guessed finding\n"), new(bytes.Buffer), options)
		})
		if err != nil {
			t.Fatalf("%s with guess: %v", reporter, err)
		}
		if !strings.Contains(output, "guessed finding") || requests.Load() != 0 {
			t.Errorf("%s with guess: output=%q requests=%d", reporter, output, requests.Load())
		}
	}
}
