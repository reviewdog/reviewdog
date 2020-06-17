package project

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
)

var (
	defaultTeeStdout io.Writer = os.Stdout
	defaultTeeStderr io.Writer = os.Stderr
)

type cmdBuilder struct {
	envs      []string
	teeStdout io.Writer
	teeStderr io.Writer
	enableTee bool
}

func newCmdBuilder(envs []string, enableTee bool) *cmdBuilder {
	return &cmdBuilder{
		envs:      envs,
		teeStdout: defaultTeeStdout,
		teeStderr: defaultTeeStderr,
		enableTee: enableTee,
	}
}

func (cb *cmdBuilder) build(ctx context.Context, command string) (*exec.Cmd, io.Reader, io.Reader, error) {
	shell := "sh"
	args := []string{"-c", command}
	if runtime.GOOS == "windows" {
		// Under Windows the executable sh is not always available
		// If running under MinGW the environment variable SHELL would be set
		SHELL := os.Getenv("SHELL")
		// Otherwise use the environment variable COMSPEC (path to cmd.exe)
		COMSPEC := os.Getenv("COMSPEC")
		if SHELL != "" {
			shell = SHELL
		} else if COMSPEC != "" {
			shell = COMSPEC
			// cmd.exe uses "/c" instead of "-c"
			args[0] = "/c"
		}
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = cb.envs
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	var (
		teeOut io.Reader = stdout
		teeErr io.Reader = stderr
	)
	if cb.enableTee {
		teeOut = io.TeeReader(stdout, cb.teeStdout)
		teeErr = io.TeeReader(stderr, cb.teeStderr)
	}
	return cmd, teeOut, teeErr, nil
}
