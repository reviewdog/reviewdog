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
