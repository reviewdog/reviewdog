package difffilter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reviewdog/reviewdog/diff"
)

func TestMode_Set(t *testing.T) {
	tests := []struct {
		value   string
		want    Mode
		wantErr bool
	}{
		{value: "", want: ModeDefault},
		{value: "default", want: ModeDefault},
		{value: "added", want: ModeAdded},
		{value: "diff_context", want: ModeDiffContext},
		{value: "file", want: ModeFile},
		{value: "unknown", wantErr: true},
	}
	for _, tt := range tests {
		var mode Mode
		err := (&mode).Set(tt.value)
		if err != nil && !tt.wantErr {
			t.Errorf("got error for %q: %v", tt.value, err)
		} else if err == nil && tt.wantErr {
			t.Errorf("want error, but got nil for %q", tt.value)
		}
		if mode != tt.want {
			t.Errorf("[value=%s] got %q, want %q", tt.value, mode.String(), tt.want.String())
		}
	}
}

const sampleDiffRoot = `--- a/sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ b/sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- a/subdir/nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ b/subdir/nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

const sampleDiffSubDir = `--- a/difffilter/sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ b/difffilter/sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- a/sample.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ b/sample.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,5 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
+b
\ No newline at end of file
`

func getCwd() string {
	cwd, _ := os.Getwd()
	return cwd
}

func cd(path string) (cleanup func()) {
	cwd := getCwd()
	os.Chdir(path)
	return func() {
		os.Chdir(cwd)
	}
}

func getDiff(t *testing.T, difftext string) []*diff.FileDiff {
	t.Helper()
	files, err := diff.ParseMultiFile(strings.NewReader(difftext))
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestDiffFilter_root(t *testing.T) {
	defer cd("..")()
	files := getDiff(t, sampleDiffRoot)
	tests := []struct {
		path         string
		lnum         int
		mode         Mode
		want         bool
		wantLnumDiff int
	}{
		{
			path:         "sample.new.txt",
			lnum:         2,
			mode:         ModeAdded,
			want:         true,
			wantLnumDiff: 3,
		},
		{
			path:         filepath.Join(getCwd(), "sample.new.txt"),
			lnum:         2,
			mode:         ModeAdded,
			want:         true,
			wantLnumDiff: 3,
		},
		{
			path:         "sample.new.txt",
			lnum:         1,
			mode:         ModeAdded,
			want:         false,
			wantLnumDiff: 0,
		},
		{
			path:         "sample.new.txt",
			lnum:         1,
			mode:         ModeDiffContext,
			want:         true,
			wantLnumDiff: 1,
		},
		{
			path:         "subdir/nonewline.new.txt",
			lnum:         3,
			mode:         ModeAdded,
			want:         true,
			wantLnumDiff: 3,
		},
		{
			path:         "sample.new.txt",
			lnum:         14,
			mode:         ModeFile,
			want:         true,
			wantLnumDiff: 0,
		},
	}
	for _, tt := range tests {
		df := New(files, 1, getCwd(), tt.mode)
		if got, gotLine := df.InDiff(tt.path, tt.lnum); got != tt.want {
			gotLnumDiff := 0
			if gotLine != nil {
				gotLnumDiff = gotLine.LnumDiff
			}
			t.Errorf("InDiff(%q, %d) = (%v, %d), want (%v, %d)",
				tt.path, tt.lnum, got, gotLnumDiff, tt.want, tt.wantLnumDiff)
		}
	}
}

func TestDiffFilter_subdir(t *testing.T) {
	// git diff (including diff from GitHub) returns path relative to a project
	// root directory (See sampleDiffSubDir), but given path from linters can be
	// relative path to current working directory.
	files := getDiff(t, sampleDiffSubDir)
	tests := []struct {
		path         string
		lnum         int
		mode         Mode
		want         bool
		wantLnumDiff int
	}{
		{
			path:         "sample.new.txt",
			lnum:         2,
			mode:         ModeAdded,
			want:         true,
			wantLnumDiff: 3,
		},
		{
			path:         "sample.new.txt",
			lnum:         2,
			mode:         ModeDefault,
			want:         true,
			wantLnumDiff: 3,
		},
		{
			path:         filepath.Join(getCwd(), "sample.new.txt"),
			lnum:         2,
			mode:         ModeAdded,
			want:         true,
			wantLnumDiff: 3,
		},
		{
			path:         "sample.new.txt",
			lnum:         5,
			mode:         ModeAdded,
			want:         false,
			wantLnumDiff: 0,
		},
	}
	for _, tt := range tests {
		df := New(files, 1, getCwd(), tt.mode)
		if got, gotLine := df.InDiff(tt.path, tt.lnum); got != tt.want {
			gotLnumDiff := 0
			if gotLine != nil {
				gotLnumDiff = gotLine.LnumDiff
			}
			t.Errorf("InDiff(%q, %d) = (%v, %d), want (%v, %d)",
				tt.path, tt.lnum, got, gotLnumDiff, tt.want, tt.wantLnumDiff)
		}
	}
}
