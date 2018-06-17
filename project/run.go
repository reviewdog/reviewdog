package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/haya14busa/reviewdog"
	"github.com/haya14busa/reviewdog/diff"
	"golang.org/x/sync/errgroup"
)

// RunAndParse runs commands and parse results. Returns map of tool name to check results.
func RunAndParse(ctx context.Context, conf *Config) (map[string][]*reviewdog.CheckResult, error) {
	results := make(map[string][]*reviewdog.CheckResult)
	// environment variables for each commands
	envs := filteredEnviron()
	var g errgroup.Group
	semaphore := make(chan int, runtime.NumCPU())
	for _, runner := range conf.Runner {
		runner := runner
		semaphore <- 1
		fname := runner.Format
		if fname == "" && len(runner.Errorformat) == 0 {
			fname = runner.Name
		}
		opt := &reviewdog.ParserOpt{FormatName: fname, Errorformat: runner.Errorformat}
		p, err := reviewdog.NewParser(opt)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, "sh", "-c", runner.Cmd)
		cmd.Env = envs
		stdout, err := cmd.StdoutPipe()
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("fail to start command: %v", err)
		}
		g.Go(func() error {
			defer func() { <-semaphore }()
			rs, err := p.Parse(io.MultiReader(stdout, stderr))
			if err != nil {
				return err
			}
			results[runner.Name] = rs
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("fail to run reviewdog: %v", err)
	}
	return results, nil
}

// Run runs reviewdog tasks based on Config.
func Run(ctx context.Context, conf *Config, c reviewdog.CommentService, d reviewdog.DiffService) error {
	results, err := RunAndParse(ctx, conf)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}
	b, err := d.Diff(ctx)
	if err != nil {
		return err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(b))
	if err != nil {
		return err
	}
	var g errgroup.Group
	for toolname, rs := range results {
		toolname := toolname
		rs := rs
		g.Go(func() error {
			return reviewdog.RunFromResult(ctx, c, rs, filediffs, d.Strip(), toolname)
		})
	}
	return g.Wait()
}

var secretEnvs = [...]string{
	"REVIEWDOG_GITHUB_API_TOKEN",
	"REVIEWDOG_GITLAB_API_TOKEN",
	"REVIEWDOG_TOKEN",
}

func filteredEnviron() []string {
	for _, name := range secretEnvs {
		defer func(name, value string) {
			if value != "" {
				os.Setenv(name, value)
			}
		}(name, os.Getenv(name))
		os.Unsetenv(name)
	}
	return os.Environ()
}
