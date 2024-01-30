package parser

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"

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
	slf := new(SarifJson)
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
		informationURI := driver.InformationURI
		baseURIs := run.OriginalURIBaseIds
		rules := map[string]SarifRule{}
		for _, rule := range driver.Rules {
			rules[rule.ID] = rule
		}
		for _, result := range run.Results {
			original, err := json.Marshal(result)
			if err != nil {
				return nil, err
			}
			message := result.Message.GetText()
			ruleID := result.RuleID
			rule := rules[ruleID]
			level := result.Level
			if level == "" {
				level = rule.DefaultConfiguration.Level
			}
			suggestionsMap := map[string][]*rdf.Suggestion{}
			for _, fix := range result.Fixes {
				for _, artifactChange := range fix.ArtifactChanges {
					suggestions := []*rdf.Suggestion{}
					path, err := artifactChange.ArtifactLocation.GetPath(baseURIs, basedir)
					if err != nil {
						// invalid path
						return nil, err
					}
					for _, replacement := range artifactChange.Replacements {
						deletedRegion := replacement.DeletedRegion
						rng := deletedRegion.GetRdfRange()
						if rng == nil {
							// No line information in fix
							continue
						}
						s := &rdf.Suggestion{
							Range: rng,
							Text:  replacement.InsertedContent.Text,
						}
						suggestions = append(suggestions, s)
					}
					suggestionsMap[path] = suggestions
				}
			}
			for _, location := range result.Locations {
				physicalLocation := location.PhysicalLocation
				artifactLocation := physicalLocation.ArtifactLocation
				path, err := artifactLocation.GetPath(baseURIs, basedir)
				if err != nil {
					// invalid path
					return nil, err
				}
				region := physicalLocation.Region
				rng := region.GetRdfRange()
				var code *rdf.Code
				if ruleID != "" {
					code = &rdf.Code{
						Value: ruleID,
						Url:   rule.HelpURI,
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

// SARIF JSON Format
//
// References:
//   - https://sarifweb.azurewebsites.net/
type SarifJson struct {
	Runs []struct {
		OriginalURIBaseIds map[string]SarifOriginalURI `json:"originalUriBaseIds"`
		Results            []struct {
			Level     string `json:"level,omitempty"`
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation SarifArtifactLocation `json:"artifactLocation,omitempty"`
					Region           SarifRegion           `json:"region,omitempty"`
				} `json:"physicalLocation,omitempty"`
			} `json:"locations"`
			Message SarifText `json:"message"`
			RuleID  string    `json:"ruleId,omitempty"`
			Fixes   []struct {
				Description     SarifText `json:"description"`
				ArtifactChanges []struct {
					ArtifactLocation SarifArtifactLocation `json:"artifactLocation,omitempty"`
					Replacements     []struct {
						DeletedRegion   SarifRegion `json:"deletedRegion"`
						InsertedContent struct {
							Text string `json:"text"`
						} `json:"insertedContent,omitempty"`
					} `json:"replacements"`
				} `json:"artifactChanges"`
			} `json:"fixes,omitempty"`
		} `json:"results"`
		Tool struct {
			Driver struct {
				FullName       string      `json:"fullName"`
				InformationURI string      `json:"informationUri"`
				Name           string      `json:"name"`
				Rules          []SarifRule `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
	} `json:"runs"`
}

type SarifOriginalURI struct {
	URI string `json:"uri"`
}

type SarifArtifactLocation struct {
	URI       string `json:"uri,omitempty"`
	URIBaseID string `json:"uriBaseId"`
	Index     int    `json:"index,omitempty"`
}

func (l *SarifArtifactLocation) GetPath(
	baseURIs map[string]SarifOriginalURI,
	basedir string,
) (string, error) {
	uri := l.URI
	baseURI := baseURIs[l.URIBaseID].URI
	if baseURI != "" {
		if u, err := url.JoinPath(baseURI, uri); err == nil {
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

type SarifText struct {
	Text     string  `json:"text,omitempty"`
	Markdown *string `json:"markdown,omitempty"`
}

func (t *SarifText) GetText() string {
	text := t.Text
	if t.Markdown != nil {
		text = *t.Markdown
	}
	return text
}

type SarifRegion struct {
	StartLine   *int `json:"startLine"`
	StartColumn *int `json:"startColumn,omitempty"`
	EndLine     *int `json:"endLine,omitempty"`
	EndColumn   *int `json:"endColumn,omitempty"`
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
func (r *SarifRegion) GetRdfRange() *rdf.Range {
	if r.StartLine == nil {
		// No line information
		return nil
	}
	startLine := *r.StartLine
	startColumn := 1 // default value of startColumn in SARIF is 1
	if r.StartColumn != nil {
		startColumn = *r.StartColumn
	}
	endLine := startLine // default value of endLine in SARIF is startLine
	if r.EndLine != nil {
		endLine = *r.EndLine
	}
	endColumn := 0 // default value of endColumn in SARIF is null (that means EOL)
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

type SarifRule struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	ShortDescription     SarifText `json:"shortDescription"`
	FullDescription      SarifText `json:"fullDescription"`
	Help                 SarifText `json:"help"`
	HelpURI              string    `json:"helpUri"`
	DefaultConfiguration struct {
		Level string `json:"level"`
		Rank  int    `json:"rank"`
	} `json:"defaultConfiguration"`
}
