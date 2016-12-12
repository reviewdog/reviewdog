package reviewdog

import (
	"fmt"
	"io"
	"strings"
)

var _ CommentService = &CommentWriter{}

type CommentWriter struct {
	w io.Writer
}

func NewCommentWriter(w io.Writer) *CommentWriter {
	return &CommentWriter{w: w}
}

func (s *CommentWriter) Post(c *Comment) error {
	_, err := fmt.Fprintln(s.w, strings.Join(c.CheckResult.Lines, "\n"))
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
// where <message> can be multiple lines.
type UnifiedCommentWriter struct {
	w io.Writer
}

func NewUnifiedCommentWriter(w io.Writer) *UnifiedCommentWriter {
	return &UnifiedCommentWriter{w: w}
}

func (mc *UnifiedCommentWriter) Post(c *Comment) error {
	s := c.Path
	if c.Lnum > 0 {
		s += fmt.Sprintf(":%d", c.Lnum)
		if c.Col > 0 {
			s += fmt.Sprintf(":%d", c.Col)
		}
	}
	s += fmt.Sprintf(": [%s] %s", c.ToolName, c.Body)
	_, err := fmt.Fprintln(mc.w, s)
	return err
}
