package reviewdog

import (
	"fmt"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

// FailLevel represents enumeration of available filter modes
type FailLevel int

const (
	// FailLevelDefault represents default mode, which means users doesn't specify
	// fail-level. Basically, it's same as FailLevelNone.
	FailLevelDefault FailLevel = iota
	FailLevelNone
	FailLevelAny
	FailLevelInfo
	FailLevelWarning
	FailLevelError
)

// String implements the flag.Value interface
func (failLevel *FailLevel) String() string {
	names := [...]string{
		"default",
		"none",
		"any",
		"info",
		"warning",
		"error",
	}
	if *failLevel < FailLevelDefault || *failLevel > FailLevelError {
		return "Unknown failLevel"
	}

	return names[*failLevel]
}

// Set implements the flag.Value interface
func (failLevel *FailLevel) Set(value string) error {
	switch value {
	case "default", "":
		*failLevel = FailLevelDefault
	case "none":
		*failLevel = FailLevelNone
	case "any":
		*failLevel = FailLevelAny
	case "info":
		*failLevel = FailLevelInfo
	case "warning":
		*failLevel = FailLevelWarning
	case "error":
		*failLevel = FailLevelError
	default:
		return fmt.Errorf("invalid failLevel name: %s", value)
	}
	return nil
}

// ShouldFail returns true if reviewdog should exit with 1 with given rdf.Severity.
func (failLevel FailLevel) ShouldFail(severity rdf.Severity) bool {
	if failLevel == FailLevelDefault || failLevel == FailLevelNone {
		return false
	}
	minSeverity := failLevel.minSeverity()
	return minSeverity == rdf.Severity_UNKNOWN_SEVERITY || severity <= minSeverity
}

func (failLevel FailLevel) minSeverity() rdf.Severity {
	switch failLevel {
	case FailLevelDefault, FailLevelNone, FailLevelAny:
		return rdf.Severity_UNKNOWN_SEVERITY
	case FailLevelInfo:
		return rdf.Severity_INFO
	case FailLevelWarning:
		return rdf.Severity_WARNING
	case FailLevelError:
		return rdf.Severity_ERROR
	default:
		return rdf.Severity_UNKNOWN_SEVERITY
	}
}
