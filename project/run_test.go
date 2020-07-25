package project

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/filter"
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

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		conf := &Config{}
		if err := Run(ctx, conf, nil, nil, nil, false, filter.ModeAdded, false); err != nil {
			t.Error(err)
		}
	})

	t.Run("errorformat error", func(t *testing.T) {
		conf := &Config{
			Runner: map[string]*Runner{
				"test": {},
			},
		}
		if err := Run(ctx, conf, nil, nil, nil, false, filter.ModeAdded, false); err == nil {
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
		if err := Run(ctx, conf, nil, nil, ds, false, filter.ModeAdded, false); err == nil {
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
		if err := Run(ctx, conf, nil, cs, ds, false, filter.ModeAdded, false); err != nil {
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
		if err := Run(ctx, conf, nil, cs, ds, false, filter.ModeAdded, false); err == nil {
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
					Cmd:         "not found",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, cs, ds, true, filter.ModeAdded, false); err == nil {
			t.Error("want error, got nil")
		} else {
			t.Log(err)
		}
		want := "sh: 1: not: not found\n"
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
		if err := Run(ctx, conf, nil, cs, ds, false, filter.ModeAdded, false); err != nil {
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
		if err := Run(ctx, conf, nil, cs, ds, true, filter.ModeAdded, false); err != nil {
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
		if err := Run(ctx, conf, map[string]bool{"test2": true}, cs, ds, false, filter.ModeAdded, false); err != nil {
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
		if err := Run(ctx, conf, map[string]bool{"hoge": true}, cs, ds, false, filter.ModeAdded, false); err == nil {
			t.Error("got no error but want runner not found error")
		}
	})
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
