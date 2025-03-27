package filter

import (
	"fmt"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/pathutil"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

// Mode represents enumeration of available filter modes
type Mode int

const (
	// ModeDefault represents default mode, which means users doesn't specify
	// filter-mode. The behavior can be changed depending on reporters/context
	// later if we want. Basically, it's same as ModeAdded because it's most safe
	// and basic mode for reporters implementation.
	ModeDefault Mode = iota
	// ModeAdded represents filtering by added/changed diff lines.
	ModeAdded
	// ModeDiffContext represents filtering by diff context.
	// i.e. changed lines +-N lines (e.g. N=3 for default git diff).
	ModeDiffContext
	// ModeFile represents filtering by changed files.
	ModeFile
	// ModeNoFilter doesn't filter out any results.
	ModeNoFilter
)

// String implements the flag.Value interface
func (mode *Mode) String() string {
* ▋
