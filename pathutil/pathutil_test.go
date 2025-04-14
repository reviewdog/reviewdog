package pathutil

import (
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

func TestNormalizePathInResults(t *testing.T) {
	cwd := "/path/to/cwd"
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
	NormalizePathInResults(results, cwd)
	for _, result := range results {
		if strings.HasPrefix(result.GetLocation().GetPath(), cwd) {
			t.Errorf("path unexpectedly contain prefix: %s", result.GetLocation().GetPath())
		}
		for _, rel := range result.GetRelatedLocations() {
			if strings.HasPrefix(rel.GetLocation().GetPath(), cwd) {
				t.Errorf("related locations path unexpectedly contain prefix: %s", rel.GetLocation().GetPath())
			}
		}
	}
}
