package reviewdog

import (
	"bytes"
	"strings"
	"testing"
)

func TestMultiCommentService_Post(t *testing.T) {
	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	w := MultiCommentService(NewRawCommentWriter(buf1), NewRawCommentWriter(buf2))

	const want = "line1\nline2"

	c := &Comment{CheckResult: &CheckResult{Lines: strings.Split(want, "\n")}}
	if err := w.Post(c); err != nil {
		t.Fatal(err)
	}

	if got := strings.Trim(buf1.String(), "\n"); got != want {
		t.Errorf("writer 1: got %v, want %v", got, want)
	}

	if got := strings.Trim(buf2.String(), "\n"); got != want {
		t.Errorf("writer 2: got %v, want %v", got, want)
	}

	if err := w.(BulkCommentService).Flash(); err != nil {
		t.Errorf("MultiCommentService implements BulkCommentService and should not return error when any services implements it: %v", err)
	}
}

type fakeBulkCommentService struct {
	BulkCommentService
	calledFlash bool
}

func (f *fakeBulkCommentService) Flash() error {
	f.calledFlash = true
	return nil
}

func TestMultiCommentService_Flash(t *testing.T) {
	f1 := &fakeBulkCommentService{}
	f2 := &fakeBulkCommentService{}
	w := MultiCommentService(f1, f2)
	if err := w.(BulkCommentService).Flash(); err != nil {
		t.Fatal(err)
	}
	if !f1.calledFlash || !f2.calledFlash {
		t.Error("MultiCommentService_Flash should run Flash() for every services")
	}
}
