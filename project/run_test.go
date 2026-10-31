package project

import (
	"bytes"
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/parser"
)

type fakeDiffService struct {
	reviewdog.DiffService
	FakeDiff func() ([]byte, error)
}

func (f *fakeDiffService) Diff(_ context.Context) ([]byte, error) {
	return f.FakeDiff()
}

func (f *fakeDiffService) Strip() int {
	return 0
}

type fakeCommentService struct {
	reviewdog.CommentService
	FakePost func(*reviewdog.Comment) error
}

func (f *fakeCommentService) Post(_ context.Context, c *reviewdog.Comment) error {
	return f.FakePost(c)
}

func (*fakeCommentService) ShouldPrependGitRelDir() bool { return false }

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		conf := &Config{}
		if err := Run(ctx, conf, nil, nil, nil, false, filter.ModeAdded, reviewdog.FailLevelNone); err != nil {
			t.Error(err)
		}
	})

	t.Run("errorformat error", func(t *testing.T) {
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {},
			},
		}
		if err := Run(ctx, conf, nil, nil, nil, false, filter.ModeAdded, reviewdog.FailLevelNone); err == nil {
			t.Error("want error, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("diff error", func(t *testing.T) {
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return nil, errors.New("err!")
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "echo 'hi'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, nil, ds, false, filter.ModeAdded, reviewdog.FailLevelNone); err == nil {
			t.Error("want error, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("cmd error with findings (not for reviewdog to exit with error)", func(t *testing.T) {
		buf := new(bytes.Buffer)
		defaultTeeStderr = buf
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "echo 'file:14:14:message'; exit 1",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, cs, ds, false, filter.ModeAdded, reviewdog.FailLevelNone); err != nil {
			t.Error(err)
		}
		want := ""
		if got := buf.String(); got != want {
			t.Errorf("got stderr %q, want %q", got, want)
		}
	})

	t.Run("unexpected cmd error (reviewdog exits with error)", func(t *testing.T) {
		buf := new(bytes.Buffer)
		defaultTeeStderr = buf
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "not found",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, cs, ds, false, filter.ModeAdded, reviewdog.FailLevelNone); err == nil {
			t.Error("want error, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("cmd error with tee", func(t *testing.T) {
		buf := new(bytes.Buffer)
		defaultTeeStderr = buf
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "not-found",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, cs, ds, true, filter.ModeAdded, reviewdog.FailLevelNone); err == nil {
			t.Error("want error, got nil")
		} else {
			t.Log(err)
		}
		var want string
		if runtime.GOOS == "darwin" {
			want = "sh: not-found: command not found\n"
		} else {
			want = "sh: 1: not-found: not found\n"
		}
		if got := buf.String(); got != want {
			t.Errorf("got stderr %q, want %q", got, want)
		}
	})

	t.Run("success", func(t *testing.T) {
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "echo 'hi'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, cs, ds, false, filter.ModeAdded, reviewdog.FailLevelNone); err != nil {
			t.Error(err)
		}
	})

	t.Run("success with tee", func(t *testing.T) {
		buf := new(bytes.Buffer)
		defaultTeeStdout = buf
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "echo 'hi'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, cs, ds, true, filter.ModeAdded, reviewdog.FailLevelNone); err != nil {
			t.Error(err)
		}
		want := "hi\n"
		if got := buf.String(); got != want {
			t.Errorf("got stdout %q, want %q", got, want)
		}
	})

	t.Run("runners", func(t *testing.T) {
		called := 0
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				called++
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test1": {
					Name:        "test1",
					Cmd:         "echo 'test1'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
				"test2": {
					Name:        "test2",
					Cmd:         "echo 'test2'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, map[string]bool{"test2": true}, cs, ds, false, filter.ModeAdded, reviewdog.FailLevelNone); err != nil {
			t.Error(err)
		}
		if called != 1 {
			t.Errorf("Diff service called %d times, want 1 time", called)
		}
	})

	t.Run("unknown runners", func(t *testing.T) {
		ds := &fakeDiffService{
			FakeDiff: func() ([]byte, error) {
				return []byte(""), nil
			},
		}
		cs := &fakeCommentService{
			FakePost: func(c *reviewdog.Comment) error {
				return nil
			},
		}
		conf := &Config{
			Runner: map[string]*Runner{
				"test1": {
					Name:        "test1",
					Cmd:         "echo 'test1'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
				"test2": {
					Name:        "test2",
					Cmd:         "echo 'test2'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, map[string]bool{"hoge": true}, cs, ds, false, filter.ModeAdded, reviewdog.FailLevelNone); err == nil {
			t.Error("got no error but want runner not found error")
		}
	})

	// #1934
	t.Run("big stderr", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("requires POSIX shell utilities")
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "yes x | head -c 131072 >&2; echo 'file:1:1:message'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}

		results, err := RunAndParse(ctx, conf, nil, "", false)
		if err != nil {
			t.Fatalf("RunAndParse failed: %v", err)
		}

		result, err := results.Load("test")
		if err != nil {
			t.Fatalf("failed to load result: %v", err)
		}
		if result.CmdErr != nil {
			t.Fatalf("unexpected command error: %v", result.CmdErr)
		}
		if len(result.Diagnostics) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(result.Diagnostics))
		}
	})

	t.Run("interleaved multiline diagnostics maintain stream affinity", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("requires POSIX shell utilities")
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "printf 'a.go:1: error A\\n  detail A\\nEND\\n'; printf 'b.go:2: error B\\n  detail B\\nEND\\n' >&2",
					Errorformat: []string{"%E%f:%l: %m", "%C  %m", "%ZEND"},
				},
			},
		}

		results, err := RunAndParse(ctx, conf, nil, "", false)
		if err != nil {
			t.Fatalf("RunAndParse failed: %v", err)
		}

		result, err := results.Load("test")
		if err != nil {
			t.Fatalf("failed to load result: %v", err)
		}
		if result.CmdErr != nil {
			t.Fatalf("unexpected command error: %v", result.CmdErr)
		}
		if len(result.Diagnostics) != 2 {
			t.Fatalf("got %d diagnostics, want 2: %#v", len(result.Diagnostics), result.Diagnostics)
		}
		if result.Diagnostics[0].Location.Path != "a.go" || result.Diagnostics[0].Message != "error A\ndetail A" {
			t.Errorf("diagnostic 0 mismatch: %#v", result.Diagnostics[0])
		}
		if result.Diagnostics[1].Location.Path != "b.go" || result.Diagnostics[1].Message != "error B\ndetail B" {
			t.Errorf("diagnostic 1 mismatch: %#v", result.Diagnostics[1])
		}
	})

	t.Run("cross stream continuation lines do not attach across stream boundaries", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("requires POSIX shell utilities")
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "printf 'a.go:1: error A\\n'; printf '  orphan continuation\\nEND\\nb.go:2: error B\\n  detail B\\nEND\\n' >&2",
					Errorformat: []string{"%E%f:%l: %m", "%C  %m", "%ZEND"},
				},
			},
		}

		results, err := RunAndParse(ctx, conf, nil, "", false)
		if err != nil {
			t.Fatalf("RunAndParse failed: %v", err)
		}

		result, err := results.Load("test")
		if err != nil {
			t.Fatalf("failed to load result: %v", err)
		}
		if len(result.Diagnostics) != 2 {
			t.Fatalf("got %d diagnostics, want 2: %#v", len(result.Diagnostics), result.Diagnostics)
		}
		if result.Diagnostics[0].Location.Path != "a.go" || result.Diagnostics[0].Location.Range.Start.Line != 1 || result.Diagnostics[0].Message != "error A" {
			t.Errorf("diagnostic 0 mismatch: %#v", result.Diagnostics[0])
		}
		if result.Diagnostics[1].Location.Path != "b.go" || result.Diagnostics[1].Location.Range.Start.Line != 2 || result.Diagnostics[1].Message != "error B\ndetail B" {
			t.Errorf("diagnostic 1 mismatch: %#v", result.Diagnostics[1])
		}
	})

	t.Run("headers on stdout and details on stderr do not attach across streams", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("requires POSIX shell utilities")
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		conf := &Config{
			Runner: map[string]*Runner{
				"test": {
					Cmd:         "printf 'a.go:1: error A\\nb.go:2: error B\\n'; printf '  detail A\\nEND\\n  detail B\\nEND\\n' >&2",
					Errorformat: []string{"%E%f:%l: %m", "%C  %m", "%ZEND"},
				},
			},
		}

		results, err := RunAndParse(ctx, conf, nil, "", false)
		if err != nil {
			t.Fatalf("RunAndParse failed: %v", err)
		}

		result, err := results.Load("test")
		if err != nil {
			t.Fatalf("failed to load result: %v", err)
		}
		if result.CmdErr != nil {
			t.Fatalf("unexpected command error: %v", result.CmdErr)
		}
		if len(result.Diagnostics) != 2 {
			t.Fatalf("got %d diagnostics, want 2: %#v", len(result.Diagnostics), result.Diagnostics)
		}
		if result.Diagnostics[0].Location.Path != "a.go" || result.Diagnostics[0].Location.Range.Start.Line != 1 || result.Diagnostics[0].Message != "error A" {
			t.Errorf("diagnostic 0 mismatch: %#v", result.Diagnostics[0])
		}
		if result.Diagnostics[1].Location.Path != "b.go" || result.Diagnostics[1].Location.Range.Start.Line != 2 || result.Diagnostics[1].Message != "error B" {
			t.Errorf("diagnostic 1 mismatch: %#v", result.Diagnostics[1])
		}
	})
}

func TestParseRunnerOutput_StructuredFallback(t *testing.T) {
	p := parser.NewCheckStyleParser()
	stdout := strings.NewReader(`<?xml version="1.0" encoding="utf-8"?><checkstyle version="4.3"><file name="main.go"><error line="5" column="1" severity="error" message="checkstyle error" source="golint" /></file></checkstyle>`)
	stderr := strings.NewReader("WARN: build cache miss\n")

	diags, err := parseRunnerOutput(p, stdout, stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diags))
	}
	if diags[0].Location.Path != "main.go" || diags[0].Message != "checkstyle error" {
		t.Errorf("unexpected diagnostic: %#v", diags[0])
	}
}

func TestFilteredEnviron(t *testing.T) {
	names := [...]string{
		"REVIEWDOG_GITHUB_API_TOKEN",
		"REVIEWDOG_GITLAB_API_TOKEN",
		"REVIEWDOG_TOKEN",
	}

	for _, name := range names {
		defer func(name, value string) {
			os.Setenv(name, value)
		}(name, os.Getenv(name))
		os.Setenv(name, "value")
	}

	filtered := filteredEnviron()
	if len(filtered) != len(os.Environ())-len(names) {
		t.Errorf("len(filtered) != len(os.Environ())-%d, %v != %v-%d", len(names), len(filtered), len(os.Environ()), len(names))
	}

	for _, kv := range filtered {
		for _, name := range names {
			if strings.HasPrefix(kv, name) && kv != name+"=" {
				t.Errorf("filtered: %v, want %v=", kv, name)
			}
		}
	}

	for _, kv := range os.Environ() {
		for _, name := range names {
			if strings.HasPrefix(kv, name) && kv != name+"=value" {
				t.Errorf("envs: %v, want %v=value", kv, name)
			}
		}
	}
}
