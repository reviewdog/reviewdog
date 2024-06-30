package reviewdog

import (
	"context"
	"fmt"
	"io"

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
