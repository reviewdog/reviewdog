	"google.golang.org/protobuf/testing/protocmp"
	"github.com/reviewdog/reviewdog/proto/rdf"
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "sample.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
				},
			},
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "sample.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
				},
			},
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "nonewline.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
				},
			},
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "nonewline.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 3}},
				},
			},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
					},
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "nonewline.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
					},
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "nonewline.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 3}},
					},
				},
	if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "sample.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
				},
			},
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "sample.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
				},
			},
			Diagnostic: &rdf.Diagnostic{
				Location: &rdf.Location{
					Path:  "sample.new.txt",
					Range: &rdf.Range{Start: &rdf.Position{Line: 3}},
				},
			},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 1}},
					},
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 2}},
					},
				},
				Diagnostic: &rdf.Diagnostic{
					Location: &rdf.Location{
						Path:  "sample.new.txt",
						Range: &rdf.Range{Start: &rdf.Position{Line: 3}},
					},
				},
	if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {