package reviewdog

import (
	"os/exec"
)

var _ DiffService = &DiffString{}

type DiffString struct {
	b     []byte
	strip int
}

func NewDiffString(diff string, strip int) DiffService {
	return &DiffString{b: []byte(diff), strip: strip}
}

func (d *DiffString) Diff() ([]byte, error) {
	return d.b, nil
}

func (d *DiffString) Strip() int {
	return d.strip
}

var _ DiffService = &DiffCmd{}

type DiffCmd struct {
	cmd   *exec.Cmd
	strip int
}

func NewDiffCmd(cmd *exec.Cmd, strip int) DiffService {
	return &DiffCmd{cmd: cmd, strip: strip}
}

func (d *DiffCmd) Diff() ([]byte, error) {
	return d.cmd.Output()
}

func (d *DiffCmd) Strip() int {
	return d.strip
}
