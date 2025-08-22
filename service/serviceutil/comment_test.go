package serviceutil

import (
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestFingerprint(t *testing.T) {
	m := rdf.Diagnostic{
		Message: "test",
		Location: &rdf.Location{
			Path: "a.go",
			Range: &rdf.Range{
				Start: &rdf.Position{
					Line:   1,
					Column: 1,
				},
				End: &rdf.Position{
					Line:   1,
					Column: 1,
				},
			},
		},
	}

	got, err := Fingerprint(&m)
	if err != nil {
		t.Fatal(err)
	}

	want := "d102792a57188ea4"
	if want != got {
		t.Errorf("Fingerprint() = %q, want %q", got, want)
	}
}

func TestBuildMetaComment(t *testing.T) {
	fprint := "d102792a57188ea4"
	toolName := "testdog"
	got := BuildMetaComment(fprint, toolName)
	want := "<!-- __reviewdog__:ChBkMTAyNzkyYTU3MTg4ZWE0Egd0ZXN0ZG9n -->"
	if got != want {
		t.Errorf("BuildMetaComment() = %q, want %q", got, want)
	}
}

func TestExtractMetaComment(t *testing.T) {
	comment := "<!-- __reviewdog__:ChBkMTAyNzkyYTU3MTg4ZWE0Egd0ZXN0ZG9n -->"

	m := ExtractMetaComment(comment)
	fingerprint := "d102792a57188ea4"
	toolName := "testdog"

	if m.Fingerprint != fingerprint {
		t.Errorf("ExtractMetaComment() = %q, want %q", m.Fingerprint, fingerprint)
	}

	if m.SourceName != toolName {
		t.Errorf("ExtractMetaComment() = %q, want %q", m.SourceName, toolName)
	}
}
