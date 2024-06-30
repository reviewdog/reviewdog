package reviewdog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/haya14busa/go-sarif/sarif"
	"github.com/reviewdog/reviewdog/proto/rdf"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ CommentService = &RawCommentWriter{}

// RawCommentWriter is comment writer which writes results to given writer
// without any formatting.
type RawCommentWriter struct {
	w io.Writer
}

func NewRawCommentWriter(w io.Writer) *RawCommentWriter {
	return &RawCommentWriter{w: w}
}

func (s *RawCommentWriter) Post(_ context.Context, c *Comment) error {
	_, err := fmt.Fprintln(s.w, c.Result.Diagnostic.OriginalOutput)
	return err
}

var _ CommentService = &UnifiedCommentWriter{}

// UnifiedCommentWriter is comment writer which writes results to given writer
// in one of following unified formats.
//
// Format:
//   - <file>: [<tool name>] <message>
//   - <file>:<lnum>: [<tool name>] <message>
//   - <file>:<lnum>:<col>: [<tool name>] <message>
//
// where <message> can be multiple lines.
type UnifiedCommentWriter struct {
	w io.Writer
}

func NewUnifiedCommentWriter(w io.Writer) *UnifiedCommentWriter {
	return &UnifiedCommentWriter{w: w}
}

func (mc *UnifiedCommentWriter) Post(_ context.Context, c *Comment) error {
	loc := c.Result.Diagnostic.GetLocation()
	s := loc.GetPath()
	start := loc.GetRange().GetStart()
	if start.GetLine() > 0 {
		s += fmt.Sprintf(":%d", start.GetLine())
		if start.GetColumn() > 0 {
			s += fmt.Sprintf(":%d", start.GetColumn())
		}
	}
	s += fmt.Sprintf(": [%s] %s", c.ToolName, c.Result.Diagnostic.GetMessage())
	_, err := fmt.Fprintln(mc.w, s)
	return err
}

var _ CommentService = &RDJSONLCommentWriter{}

// RDJSONLCommentWriter
type RDJSONLCommentWriter struct {
	w io.Writer
}

func NewRDJSONLCommentWriter(w io.Writer) *RDJSONLCommentWriter {
	return &RDJSONLCommentWriter{w: w}
}

func (cw *RDJSONLCommentWriter) Post(_ context.Context, c *Comment) error {
	if c.ToolName != "" && c.Result.Diagnostic.GetSource().GetName() == "" {
		c.Result.Diagnostic.Source = &rdf.Source{
			Name: c.ToolName,
		}
	}
	b, err := protojson.MarshalOptions{
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		Multiline:         false,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
	}.Marshal(c.Result.Diagnostic)
	if err != nil {
		return err
	}
	if _, err = cw.w.Write(b); err != nil {
		return err
	}
	if _, err := cw.w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

var _ CommentService = &RDJSONCommentWriter{}

// RDJSONCommentWriter
type RDJSONCommentWriter struct {
	w        io.Writer
	comments []*Comment
	toolName string
}

func NewRDJSONCommentWriter(w io.Writer, toolName string) *RDJSONCommentWriter {
	return &RDJSONCommentWriter{w: w, toolName: toolName}
}

func (cw *RDJSONCommentWriter) Post(_ context.Context, c *Comment) error {
	cw.comments = append(cw.comments, c)
	return nil
}

func (cw *RDJSONCommentWriter) Flush(_ context.Context) error {
	result := &rdf.DiagnosticResult{
		Source: &rdf.Source{
			Name: cw.toolName,
		},
		Diagnostics: make([]*rdf.Diagnostic, 0, len(cw.comments)),
	}
	for _, c := range cw.comments {
		result.Diagnostics = append(result.Diagnostics, c.Result.Diagnostic)
	}
	b, err := protojson.MarshalOptions{
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		Multiline:         true,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
	}.Marshal(result)
	if err != nil {
		return err
	}
	if _, err = cw.w.Write(b); err != nil {
		return err
	}
	if _, err := cw.w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

var _ CommentService = &SARIFCommentWriter{}

// SARIFCommentWriter
type SARIFCommentWriter struct {
	w        io.Writer
	comments []*Comment
	toolName string
}

func NewSARIFCommentWriter(w io.Writer, toolName string) *SARIFCommentWriter {
	return &SARIFCommentWriter{w: w, toolName: toolName}
}

func (cw *SARIFCommentWriter) Post(_ context.Context, c *Comment) error {
	cw.comments = append(cw.comments, c)
	return nil
}

func (cw *SARIFCommentWriter) Flush(_ context.Context) error {
	run := sarif.Run{
		Tool: sarif.Tool{
			Driver: sarif.ToolComponent{
				Name: cw.toolName,
			},
		},
	}
	rules := make(map[string]sarif.ReportingDescriptor)
	for _, c := range cw.comments {
		result := sarif.Result{
			Message: sarif.Message{
				Text: sarif.String(c.Result.Diagnostic.Message),
			},
		}
		if code := c.Result.Diagnostic.GetCode(); code.GetValue() != "" {
			result.RuleID = sarif.String(code.GetValue())
			rules[code.GetValue()] = sarif.ReportingDescriptor{
				ID:      code.GetValue(),
				HelpURI: sarif.String(code.GetUrl()),
			}
		}
		level := severity2level(c.Result.Diagnostic.GetSeverity())
		if level != sarif.None {
			result.Level = &level
		}
		artifactLoc := sarif.ArtifactLocation{
			URI: sarif.String(c.Result.Diagnostic.GetLocation().GetPath()),
		}
		result.Locations = []sarif.Location{{
			PhysicalLocation: &sarif.PhysicalLocation{
				ArtifactLocation: &artifactLoc,
				Region:           range2region(c.Result.Diagnostic.GetLocation().GetRange()),
			},
		}}
		if len(c.Result.Diagnostic.GetSuggestions()) > 0 {
			result.Fixes = make([]sarif.Fix, 0)
			for _, suggestion := range c.Result.Diagnostic.GetSuggestions() {
				result.Fixes = append(result.Fixes, sarif.Fix{
					ArtifactChanges: []sarif.ArtifactChange{
						{
							ArtifactLocation: artifactLoc,
							Replacements: []sarif.Replacement{{
								DeletedRegion: *range2region(suggestion.GetRange()),
								InsertedContent: &sarif.ArtifactContent{
									Text: sarif.String(suggestion.GetText()),
								},
							}},
						},
					},
				})
			}
		}
		if len(c.Result.Diagnostic.GetRelatedLocations()) > 0 {
			result.RelatedLocations = make([]sarif.Location, 0)
			for _, relLoc := range c.Result.Diagnostic.GetRelatedLocations() {
				result.RelatedLocations = append(result.RelatedLocations, sarif.Location{
					PhysicalLocation: &sarif.PhysicalLocation{
						ArtifactLocation: &sarif.ArtifactLocation{
							URI: sarif.String(relLoc.GetLocation().GetPath()),
						},
						Region: range2region(relLoc.GetLocation().GetRange()),
					},
					Message: &sarif.Message{
						Text: sarif.String(relLoc.Message),
					},
				})
			}
		}
		run.Results = append(run.Results, result)
	}
	slf := sarif.NewSarif()
	run.Tool.Driver.Rules = make([]sarif.ReportingDescriptor, 0, len(rules))
	for _, r := range rules {
		run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, r)
	}
	slf.Runs = []sarif.Run{run}
	encoder := json.NewEncoder(cw.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(slf)
}

func range2region(rng *rdf.Range) *sarif.Region {
	region := &sarif.Region{}
	start := rng.GetStart()
	end := rng.GetEnd()
	if start.GetLine() > 0 {
		region.StartLine = sarif.Int64(int64(start.GetLine()))
	}
	if start.GetColumn() > 0 {
		// Column is not usually unicodeCodePoints, but let's just keep it
		// as is...
		region.StartColumn = sarif.Int64(int64(start.GetColumn()))
	}
	if end.GetLine() > 0 {
		region.EndLine = sarif.Int64(int64(end.GetLine()))
	}
	if end.GetColumn() > 0 {
		region.EndColumn = sarif.Int64(int64(end.GetColumn()))
	}
	return region
}

func severity2level(s rdf.Severity) sarif.Level {
	switch s {
	case rdf.Severity_ERROR:
		return sarif.Error
	case rdf.Severity_WARNING:
		return sarif.Warning
	default:
		return sarif.None
	}
}
