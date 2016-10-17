package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/haya14busa/errorformat"
	"github.com/haya14busa/watchdogs"
	"github.com/mattn/go-shellwords"
)

const usageMessage = "" +
	`Usage: watchdogs [flags]
`

// flags
var (
	diffCmd   string
	diffStrip int
	efms      strslice
)

func init() {
	flag.StringVar(&diffCmd, "diff", "", "diff command for filitering checker results")
	flag.IntVar(&diffStrip, "strip", 1, "strip NUM leading components from diff file names (equivalent to `patch -p`) (default is 1 for git diff)")
	flag.Var(&efms, "efm", "list of errorformat")
}

func usage() {
	fmt.Fprintln(os.Stderr, usageMessage)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if err := run(os.Stdin, os.Stdout, diffCmd, diffStrip, efms); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer, diffCmd string, diffStrip int, efms []string) error {
	d, err := diffService(diffCmd, diffStrip)
	if err != nil {
		return err
	}
	p, err := efmParser(efms)
	if err != nil {
		return err
	}
	c := watchdogs.NewCommentWriter(w)
	app := watchdogs.NewWatchdogs(p, c, d)
	return app.Run(r)
}

func efmParser(efms []string) (watchdogs.Parser, error) {
	efm, err := errorformat.NewErrorformat(efms)
	if err != nil {
		return nil, err
	}
	return watchdogs.NewErrorformatParser(efm), nil
}

func diffService(s string, strip int) (watchdogs.DiffService, error) {
	cmds, err := shellwords.Parse(s)
	if err != nil {
		return nil, err
	}
	if len(cmds) < 1 {
		return nil, errors.New("diff command is empty")
	}
	cmd := exec.Command(cmds[0], cmds[1:]...)
	d := watchdogs.NewDiffCmd(cmd, strip)
	return d, nil
}

type strslice []string

func (ss *strslice) String() string {
	return fmt.Sprintf("%v", *ss)
}

func (ss *strslice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}
