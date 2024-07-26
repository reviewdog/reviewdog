package reviewdog

import (
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestShouldFail(t *testing.T) {
	tests := []struct {
		failLevel FailLevel
		severity  rdf.Severity
		want      bool
	}{
		{
			failLevel: FailLevelDefault, severity: rdf.Severity_ERROR, want: false,
		},
		{
			failLevel: FailLevelDefault, severity: rdf.Severity_WARNING, want: false,
		},
		{
			failLevel: FailLevelDefault, severity: rdf.Severity_INFO, want: false,
		},
		{
			failLevel: FailLevelDefault, severity: rdf.Severity_UNKNOWN_SEVERITY, want: false,
		},
		{
			failLevel: FailLevelNone, severity: rdf.Severity_ERROR, want: false,
		},
		{
			failLevel: FailLevelError, severity: rdf.Severity_ERROR, want: true,
		},
		{
			failLevel: FailLevelError, severity: rdf.Severity_WARNING, want: false,
		},
		{
			failLevel: FailLevelError, severity: rdf.Severity_INFO, want: false,
		},
		{
			failLevel: FailLevelError, severity: rdf.Severity_UNKNOWN_SEVERITY, want: true,
		},
		{
			failLevel: FailLevelWarning, severity: rdf.Severity_ERROR, want: true,
		},
		{
			failLevel: FailLevelWarning, severity: rdf.Severity_WARNING, want: true,
		},
		{
			failLevel: FailLevelWarning, severity: rdf.Severity_INFO, want: false,
		},
		{
			failLevel: FailLevelWarning, severity: rdf.Severity_UNKNOWN_SEVERITY, want: true,
		},
		{
			failLevel: FailLevelInfo, severity: rdf.Severity_ERROR, want: true,
		},
		{
			failLevel: FailLevelInfo, severity: rdf.Severity_WARNING, want: true,
		},
		{
			failLevel: FailLevelInfo, severity: rdf.Severity_INFO, want: true,
		},
		{
			failLevel: FailLevelInfo, severity: rdf.Severity_UNKNOWN_SEVERITY, want: true,
		},
	}
	for _, tt := range tests {
		if got := tt.failLevel.ShouldFail(tt.severity); got != tt.want {
			t.Errorf("FailLevel(%s).ShouldFail(%s) = %v, want %v", tt.failLevel.String(),
				tt.severity, got, tt.want)
		}
	}
}
