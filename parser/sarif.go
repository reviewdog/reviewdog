package parser

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/haya14busa/go-sarif/sarif"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ Parser = &SarifParser{}

// SarifParser is sarif parser.
type SarifParser struct{}

// NewSarifParser returns a new SarifParser.
func NewSarifParser() Parser {
	return &SarifParser{}
}

func (p *SarifParser) Parse(r io.Reader) ([]*rdf.Diagnostic, error) {
	slf := new(sarif.Sarif)
	if err := json.NewDecoder(r).Decode(slf); err != nil {
		return nil, err
	}
	var ds []*rdf.Diagnostic
	basedir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if root, err := serviceutil.GetGitRoot(); err == nil {
		basedir = root
	}
	for _, run := range slf.Runs {
		tool := run.Tool
		driver := tool.Driver
		name := driver.Name
		informationURI := ""
		if driver.InformationURI != nil {
			informationURI = *driver.InformationURI
		}
		baseURIs := run.OriginalURIBaseIDS
		rules := map[string]sarif.ReportingDescriptor{}
		for _, rule := range driver.Rules {
			rules[rule.ID] = rule
		}
		for _, result := range run.Results {
			original, err := json.Marshal(result)
			if err != nil {
				return nil, err
			}
			message := getText(result.Message)
			rule := sarif.ReportingDescriptor{}
			ruleID := ""
			if result.RuleID != nil {
				ruleID = *result.RuleID
			}
			rule = rules[ruleID]
			level := ""
			if result.Level != nil {
				level = string(*result.Level)
			} else if rule.DefaultConfiguration != nil && rule.DefaultConfiguration.Level != nil {
				level = string(*rule.DefaultConfiguration.Level)
			}
			suggestionsMap := map[string][]*rdf.Suggestion{}
			for _, fix := range result.Fixes {
				for _, artifactChange := range fix.ArtifactChanges {
					suggestions := []*rdf.Suggestion{}
					path, err := getPath(artifactChange.ArtifactLocation, baseURIs, basedir)
					if err != nil {
						// invalid path
						return nil, err
					}
					for _, replacement := range artifactChange.Replacements {
						deletedRegion := replacement.DeletedRegion
						rng := getRdfRange(deletedRegion)
						if rng == nil || replacement.InsertedContent.Text == nil {
							// No line information in fix
							continue
						}
						s := &rdf.Suggestion{
							Range: rng,
							Text:  *replacement.InsertedContent.Text,
						}
						suggestions = append(suggestions, s)
					}
					suggestionsMap[path] = suggestions
				}
			}
			for _, location := range result.Locations {
				physicalLocation := location.PhysicalLocation
				artifactLocation := physicalLocation.ArtifactLocation
				loc := sarif.ArtifactLocation{}
				if artifactLocation != nil {
					loc = *artifactLocation
				}
				path, err := getPath(loc, baseURIs, basedir)
				if err != nil {
					// invalid path
					return nil, err
				}
				region := sarif.Region{}
				if physicalLocation.Region != nil {
					region = *physicalLocation.Region
				}
				rng := getRdfRange(region)
				var code *rdf.Code
				if ruleID != "" {
					code = &rdf.Code{
						Value: ruleID,
					}
					if rule.HelpURI != nil {
						code.Url = *rule.HelpURI
					}
				}
				d := &rdf.Diagnostic{
					Message: message,
					Location: &rdf.Location{
						Path:  path,
						Range: rng,
					},
					Severity: severity(level),
					Source: &rdf.Source{
						Name: name,
						Url:  informationURI,
					},
					Code:           code,
					Suggestions:    suggestionsMap[path],
					OriginalOutput: string(original),
				}
				ds = append(ds, d)
			}
		}
	}
	return ds, nil
}

func getPath(
	l sarif.ArtifactLocation,
	baseURIs map[string]sarif.ArtifactLocation,
	basedir string,
) (string, error) {
	uri := ""
	if l.URI != nil {
		uri = *l.URI
	}
	urlBaseID := ""
	if l.URIBaseID != nil {
		urlBaseID = *l.URIBaseID
	}
	baseURI := baseURIs[urlBaseID].URI
	if baseURI != nil && *baseURI != "" {
		if u, err := url.JoinPath(*baseURI, uri); err == nil {
			uri = u
		}
	}
	parse, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	path := parse.Path
	if relpath, err := filepath.Rel(basedir, path); err == nil {
		path = relpath
	}
	return path, nil
}

func getText(msg sarif.Message) string {
	text := ""
	if msg.Text != nil {
		text = *msg.Text
	}
	if msg.Markdown != nil {
		text = *msg.Markdown
	}
	return text
}

// convert SARIF Region to RDF Range
//
// * Supported SARIF: Line + Column Text region
// * Not supported SARIF: Offset + Length Text region ("charOffset", "charLength"), Binary region
//
// example text:
//
//	abc\n
//	def\n
//
// region: "abc"
// SARIF: { "startLine": 1 }
//
//	= { "startLine": 1, "startColumn": 1, "endLine": 1, "endColumn": null }
//
// -> RDF: { "start": { "line": 1 } }
//
// region: "bc"
// SARIF: { "startLine": 1, "startColumn": 2 }
//
//	= { "startLine": 1, "startColumn": 2, "endLine": 1, "endColumn": null }
//
// -> RDF: { "start": { "line": 1, "column": 2 }, "end": { "line": 1 } }
//
// region: "a"
// SARIF: { "startLine": 1, "endColumn": 2 }
//
//	= { "startLine": 1, "startColumn": 1, "endLine": 1, "endColumn": 2 }
//
// -> RDF: { "start": { "line": 1 }, "end": { "column": 2 } }
//
//	= { "start": { "line": 1 }, "end": { "line": 1, "column": 2 } }
//
// region: "b"
// SARIF: { "startLine": 1, "startColumn": 2, "endColumn": 3 }
//
//	= { "startLine": 1, "startColumn": 2, "endLine": 1, "endColumn": 3 }
//
// -> RDF: { "start": { "line": 1, "column": 2 }, "end": { "column": 3 } }
//
//	= { "start": { "line": 1, "column": 2 }, "end": { "line": 1, column": 3 } }
//
// region: "abc\ndef"
// SARIF: { "startLine": 1, "endLine": 2 }
//
//	= { "startLine": 1, "startColumn": 1, "endLine": 2, "endColumn": null }
//
// -> RDF: { "start": { "line": 1 }, "end": { "line": 2 } }
//
// region: "abc\n"
// SARIF: { "startLine": 1, "endLine": 2, "endColumn": 1 }
//
//	= { "startLine": 1, "startColumn": 1, "endLine": 2, "endColumn": 1 }
//
// -> RDF: { "start": { "line": 1 }, "end": { "line": 2, "column": 1 } }
//
// zero width region: "{■}abc"
// SARIF: { "startLine": 1, "endColumn": 1 }
//
//	= { "startLine": 1, "startColumn": 1, "endLine": 1, "endColumn": 1 }
//
// -> RDF: { "start": { "line": 1, "column": 1 } }
//
//	= { "start": { "line": 1, "column": 1 }, "end": { "line": 1, "column": 1 } }
func getRdfRange(r sarif.Region) *rdf.Range {
	if r.StartLine == nil {
		// No line information
		return nil
	}
	startLine := *r.StartLine
	var startColumn int64 = 1 // default value of startColumn in SARIF is 1
	if r.StartColumn != nil {
		startColumn = *r.StartColumn
	}
	endLine := startLine // default value of endLine in SARIF is startLine
	if r.EndLine != nil {
		endLine = *r.EndLine
	}
	var endColumn int64 = 0 // default value of endColumn in SARIF is null (that means EOL)
	if r.EndColumn != nil {
		endColumn = *r.EndColumn
	}
	var end *rdf.Position
	if startLine == endLine && startColumn == endColumn {
		// zero width region
		end = &rdf.Position{
			Line:   int32(endLine),
			Column: int32(endColumn),
		}
	} else {
		// not zero width region
		if startColumn == 1 {
			// startColumn = 1 is default value, then omit it from result
			startColumn = 0
		}
		if startLine != endLine {
			// when multi line region, End property must be provided
			end = &rdf.Position{
				Line:   int32(endLine),
				Column: int32(endColumn),
			}
		} else {
			// when single line region
			if startColumn == 0 && endColumn == 0 {
				// if single whole line region, no End properties are needed
			} else {
				// otherwise, End property is needed
				end = &rdf.Position{
					Line:   int32(endLine),
					Column: int32(endColumn),
				}
			}
		}
	}
	rng := &rdf.Range{
		Start: &rdf.Position{
			Line:   int32(startLine),
			Column: int32(startColumn),
		},
		End: end,
	}
	return rng
}
