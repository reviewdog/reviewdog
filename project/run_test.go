package project

import (
	"context"
	"errors"
	"testing"

	"github.com/haya14busa/reviewdog"
)

type fakeDiffService struct {
	reviewdog.DiffService
	FakeDiff func() ([]byte, error)
}

func (f *fakeDiffService) Diff() ([]byte, error) {
	return f.FakeDiff()
}

func (f *fakeDiffService) Strip() int {
	return 0
}

type fakeCommentService struct {
	reviewdog.CommentService
	FakePost func(*reviewdog.Comment) error
}

func (f *fakeCommentService) Post(c *reviewdog.Comment) error {
	return f.FakePost(c)
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		conf := &Config{}
		if err := Run(ctx, conf, nil, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("erorformat error", func(t *testing.T) {
		conf := &Config{
			Runner: map[string]*Runner{
				"test": &Runner{},
			},
		}
		if err := Run(ctx, conf, nil, nil); err == nil {
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
				"test": &Runner{
					Cmd:         "not found",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, nil, ds); err == nil {
			t.Error("want error, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("no cmd error (not for reviewdog to exit with error)", func(t *testing.T) {
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
				"test": &Runner{
					Cmd:         "not found",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, cs, ds); err != nil {
			t.Error(err)
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
				"test": &Runner{
					Cmd:         "echo 'hi'",
					Errorformat: []string{`%f:%l:%c:%m`},
				},
			},
		}
		if err := Run(ctx, conf, cs, ds); err != nil {
			t.Error(err)
		}
	})

}
