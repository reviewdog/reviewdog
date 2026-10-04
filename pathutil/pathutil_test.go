package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestNormalizePathInResults(t *testing.T) {
	cwd := "/path/to/cwd"
	gitRelDir := "cwd"
	results := []*rdf.Diagnostic{
		{
			Location: &rdf.Location{
				Path: cwd + "/" + "sample_1_abs.txt",
			},
		},
		{
			Location: &rdf.Location{
				Path: "sample_2_rel.txt",
			},
		},
		{
			RelatedLocations: []*rdf.RelatedLocation{
				{
					Location: &rdf.Location{
						Path: cwd + "/" + "sample_related_1_abs.txt",
					},
				},
				{
					Location: &rdf.Location{
						Path: "sample_related_2_rel.txt",
					},
				},
			},
		},
	}
	NormalizePathInResults(results, cwd, gitRelDir)
	for _, result := range results {
		locPath := result.GetLocation().GetPath()
		if strings.HasPrefix(locPath, cwd) {
			t.Errorf("path unexpectedly contain prefix: %s", locPath)
		}
		if locPath != "" && !strings.HasPrefix(locPath, gitRelDir) {
			t.Errorf("path unexpectedly does not contain git rel dir prefix: %s", locPath)
		}
		for _, rel := range result.GetRelatedLocations() {
			relPath := rel.GetLocation().GetPath()
			if strings.HasPrefix(relPath, cwd) {
				t.Errorf("related locations path unexpectedly contain prefix: %s", relPath)
			}
			if relPath != "" && !strings.HasPrefix(relPath, gitRelDir) {
				t.Errorf("path unexpectedly does not contain git rel dir prefix: %s", relPath)
			}
		}
	}
}

func TestNormalizePathCaseInsensitiveWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path comparison")
	}

	workdir := filepath.Join(`C:\Reviewdog`, "Repo")
	tests := []struct {
		name string
		path string
	}{
		{
			name: "lowercase drive letter",
			path: strings.ToLower(filepath.VolumeName(workdir)) + filepath.Join(string(filepath.Separator), "Reviewdog", "Repo", "src", "main.go"),
		},
		{
			name: "different directory casing",
			path: strings.ToLower(filepath.Join(workdir, "src", "main.go")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := NormalizePath(tt.path, workdir, ""), "src/main.go"; got != want {
				t.Fatalf("NormalizePath() = %q, want %q", got, want)
			}
		})
	}
}

func TestIsOutsideWorkdir(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		outside bool
	}{
		{name: "inside workdir", relPath: filepath.Join("src", "main.go"), outside: false},
		{name: "parent directory", relPath: "..", outside: true},
		{name: "child of parent directory", relPath: filepath.Join("..", "other", "main.go"), outside: true},
		{name: "sibling with similar prefix", relPath: filepath.Join("..", "cwd2", "main.go"), outside: true},
		{name: "name beginning with two dots", relPath: filepath.Join("..foo", "main.go"), outside: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOutsideWorkdir(tt.relPath); got != tt.outside {
				t.Fatalf("isOutsideWorkdir(%q) = %t, want %t", tt.relPath, got, tt.outside)
			}
		})
	}
}
