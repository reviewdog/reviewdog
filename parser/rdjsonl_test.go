package parser

import (
	"strings"
	"testing"
)

func TestRDJSONLParser(t *testing.T) {
	const sample = `{"source":{"name":"deadcode"},"message":"'unused' is unused","location":{"path":"testdata/main.go","range":{"start":{"line":18,"column":6}}}}
{"source":{"name":"deadcode"},"message":"'unused2' is unused","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}}
{"source":{"name":"errcheck"},"message":"Error return value of 'os.Open' is not checked","location":{"path":"testdata/main.go","range":{"start":{"line":15,"column":9}}}}
{"source":{"name":"ineffassign"},"message":"ineffectual assignment to 'x'","location":{"path":"testdata/main.go","range":{"start":{"line":12,"column":2}}}}
{"source":{"name":"govet"},"message":"printf: Sprintf format %d reads arg #1, but call has 0 args","location":{"path":"testdata/main.go","range":{"start":{"line":13,"column":2}}}}
{"source":{"name":"severity-test"},"message":"severity test (string)","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}, "severity": "WARNING"}
{"source":{"name":"severity-test"},"message":"severity test (number)","location":{"path":"testdata/main.go","range":{"start":{"line":24,"column":6}}}, "severity": "WARNING"}`
	sampleLines := strings.Split(sample, "\n")
	p := NewRDJSONLParser()
	diagnostics, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		t.Error(err)
	}
	for i, d := range diagnostics {
		if got, want := d.GetOriginalOutput(), sampleLines[i]; got != want {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}
