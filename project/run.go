package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/filter"
	"github.com/reviewdog/reviewdog/parser"
)

// RunAndParse runs commands and parse results. Returns map of tool name to check results.
func RunAndParse(ctx context.Context, conf *Config, runners map[string]bool, defaultLevel string, teeMode bool) (*reviewdog.ResultMap, error) {
	var results reviewdog.ResultMap
	// environment variables for each commands
	envs := filteredEnviron()
	cmdBuilder := newCmdBuilder(envs, teeMode)
	var usedRunners []string
	var g errgroup.Group
	semaphoreNum := runtime.NumCPU()
	if teeMode {
		semaphoreNum = 1
	}
	semaphore := make(chan int, semaphoreNum)
	for key, runner := range conf.Runner {
		runner := runner
		runnerName := getRunnerName(key, runner)
		if len(runners) != 0 && !runners[runnerName] {
			continue // Skip this runner.
		}
		usedRunners = append(usedRunners, runnerName)
		semaphore <- 1
		log.Printf("reviewdog: [start]\trunner=%s", runnerName)
		fname := runner.Format
		if fname == "" && len(runner.Errorformat) == 0 {
			fname = runnerName
		}
		opt := &parser.Option{FormatName: fname, Errorformat: runner.Errorformat}
		p, err := parser.New(opt)
		if err != nil {
			return nil, err
		}
		cmd, stdout, stderr, err := cmdBuilder.build(ctx, runner.Cmd)
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("fail to start command: %w", err)
		}
		g.Go(func() error {
			defer func() { <-semaphore }()
			diagnostics, err := p.Parse(io.MultiReader(stdout, stderr))
			if err != nil {
				return err
			}
			level := runner.Level
			if level == "" {
				level = defaultLevel
			}
			cmdErr := cmd.Wait()
			results.Store(runnerName, &reviewdog.Result{
				Name:        runnerName,
				Level:       level,
				Diagnostics: diagnostics,
				CmdErr:      cmdErr,
			})
			msg := fmt.Sprintf("reviewdog: [finish]\trunner=%s", runnerName)
			if cmdErr != nil {
				msg += fmt.Sprintf("\terror=%v", cmdErr)
			}
			log.Println(msg)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("fail to run reviewdog: %w", err)
	}
	if err := checkUnknownRunner(runners, usedRunners); err != nil {
		return nil, err
	}
	return &results, nil
}

// Run runs reviewdog tasks based on Config.
func Run(ctx context.Context, conf *Config, runners map[string]bool, c reviewdog.CommentService, d reviewdog.DiffService, teeMode bool, filterMode filter.Mode, failOnError bool) error {
	results, err := RunAndParse(ctx, conf, runners, "", teeMode) // Level is not used.
	if err != nil {
		return err
	}
	if results.Len() == 0 {
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
	results.Range(func(toolname string, result *reviewdog.Result) {
		ds := result.Diagnostics
		g.Go(func() error {
			if err := result.CheckUnexpectedFailure(); err != nil {
				return err
			}
			return reviewdog.RunFromResult(ctx, c, ds, filediffs, d.Strip(), toolname, filterMode, failOnError)
		})
	})
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

func checkUnknownRunner(specifiedRunners map[string]bool, usedRunners []string) error {
	if len(specifiedRunners) == 0 {
		return nil
	}
	for _, r := range usedRunners {
		delete(specifiedRunners, r)
	}
	var rs []string
	for r := range specifiedRunners {
		rs = append(rs, r)
	}
	if len(specifiedRunners) != 0 {
		return fmt.Errorf("runner not found: [%s]", strings.Join(rs, ","))
	}
	return nil
}

func getRunnerName(key string, runner *Runner) string {
	if runner.Name != "" {
		return runner.Name
	}
	return key
}
