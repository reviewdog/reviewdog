package pathutil

import (
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
