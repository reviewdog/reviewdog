package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/haya14busa/reviewdog"
)

// Run runs reviewdog tasks based on Config.
func Run(ctx context.Context, conf *Config, c reviewdog.CommentService, d reviewdog.DiffService, verbose bool) error {
	// environment variables for each commands
	envs := filteredEnviron()
	var g errgroup.Group
	semaphore := make(chan int, runtime.NumCPU())
	for _, runner := range conf.Runner {
		semaphore <- 1
		cmdName := runner.Name
		fname := runner.Format
		if fname == "" && len(runner.Errorformat) == 0 {
			fname = cmdName
		}
		opt := &reviewdog.ParserOpt{FormatName: fname, Errorformat: runner.Errorformat}
		p, err := reviewdog.NewParser(opt)
		if err != nil {
			return err
		}
		rd := reviewdog.NewReviewdog(cmdName, p, c, d)
		cmd := exec.CommandContext(ctx, "sh", "-c", runner.Cmd)
		cmd.Env = envs
		stdout, err := cmd.StdoutPipe()
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		log.Printf("Start\t%q", cmdName)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("fail to start command: %v", err)
		}
		g.Go(func() error {
			errBuf := new(bytes.Buffer)
			teedStderr := io.TeeReader(stderr, errBuf)
			defer func() {
				if verbose {
					log.Printf("Finish\t%q: stderr:\n%s", cmdName, errBuf.String())
				}
				<-semaphore
			}()
			return rd.Run(ctx, io.MultiReader(stdout, teedStderr))
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("fail to run reviewdog: %v", err)
	}
	return nil
}

var secretEnvs = [...]string{
	"REVIEWDOG_GITHUB_API_TOKEN",
}

func filteredEnviron() []string {
	for _, name := range secretEnvs {
		defer os.Setenv(name, os.Getenv(name))
		os.Unsetenv(name)
	}
	return os.Environ()
}
