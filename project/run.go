package project

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/nakatanakatana/reviewdog"
)

// Run runs reviewdog tasks based on Config.
func Run(ctx context.Context, conf *Config, c reviewdog.CommentService, d reviewdog.DiffService) error {
	// environment variables for each commands
	envs := filteredEnviron()
	var g errgroup.Group
	semaphore := make(chan int, runtime.NumCPU())
	for _, runner := range conf.Runner {
		semaphore <- 1
		fname := runner.Format
		if fname == "" && len(runner.Errorformat) == 0 {
			fname = runner.Name
		}
		opt := &reviewdog.ParserOpt{FormatName: fname, Errorformat: runner.Errorformat}
		p, err := reviewdog.NewParser(opt)
		if err != nil {
			return err
		}
		rd := reviewdog.NewReviewdog(runner.Name, p, c, d)
		cmd := exec.CommandContext(ctx, "sh", "-c", runner.Cmd)
		cmd.Env = envs
		stdout, err := cmd.StdoutPipe()
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("fail to start command: %v", err)
		}
		g.Go(func() error {
			defer func() { <-semaphore }()
			return rd.Run(ctx, io.MultiReader(stdout, stderr))
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("fail to run reviewdog: %v", err)
	}
	return nil
}

var secretEnvs = [...]string{
	"REVIEWDOG_GITHUB_API_TOKEN",
	"REVIEWDOG_GITLAB_API_TOKEN",
}

func filteredEnviron() []string {
	for _, name := range secretEnvs {
		defer os.Setenv(name, os.Getenv(name))
		os.Unsetenv(name)
	}
	return os.Environ()
}
