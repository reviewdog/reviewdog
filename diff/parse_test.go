package diff

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseMultiFile(t *testing.T) {
	files, err := filepath.Glob("testdata/*.diff")
	if err != nil {
		t.Fatal(err)
	}
	for _, fname := range files {
		t.Log(fname)
		f, err := os.Open(fname)
		if err != nil {
			t.Fatal(err)
		}
		difffiles, err := ParseMultiFile(f)
		if err != nil {
			t.Errorf("%v: %v", fname, err)
		}
		for _, difffile := range difffiles {
			_ = difffile
			// t.Logf("%#v", difffile)
		}
		f.Close()
	}
}

func TestFileParser_Parse(t *testing.T) {
	tests := []struct {
		in   string
		want *FileDiff
	}{
		{
			in:   "",
			want: nil,
		},
		{
			in: `diff --git a/empty.txt b/empty.txtq
deleted file mode 100644
index e69de29..0000000
`,
			want: &FileDiff{
				Extended: []string{
					"diff --git a/empty.txt b/empty.txtq",
					"deleted file mode 100644",
					"index e69de29..0000000",
				},
			},
		},
		{
			in: `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
`,
			want: &FileDiff{
				PathOld: "sample.old.txt",
				PathNew: "sample.new.txt",
				TimeOld: "2016-10-13 05:09:35.820791185 +0900",
				TimeNew: "2016-10-13 05:15:26.839245048 +0900",
				Hunks: []*Hunk{
					{
						StartLineOld: 1, LineLengthOld: 3, StartLineNew: 1, LineLengthNew: 4,
						Lines: []*Line{
							{Type: 0, Content: "unchanged, contextual line", LnumDiff: 1, LnumOld: 1, LnumNew: 1},
							{Type: 1, Content: "deleted line", LnumDiff: 2, LnumOld: 2, LnumNew: 0},
							{Type: 1, Content: "added line", LnumDiff: 3, LnumOld: 0, LnumNew: 2},
							{Type: 1, Content: "added line", LnumDiff: 4, LnumOld: 0, LnumNew: 3},
							{Type: 0, Content: "unchanged, contextual line", LnumDiff: 5, LnumOld: 3, LnumNew: 4},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		p := &fileParser{r: bufio.NewReader(strings.NewReader(tt.in))}
		got, err := p.Parse()
		if err != nil {
			t.Errorf("got error %v for in:\n %v", err, tt.in)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("fileParser.Parse() = %#v, want %#v\nin: %v", got, tt.want, tt.in)
		}
	}
}

func TestParseFileHeader(t *testing.T) {
	tests := []struct {
		in        string
		filename  string
		timestamp string
	}{
		{
			in: "--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900",
			filename:  "sample.old.txt",
			timestamp: "2016-10-13 05:09:35.820791185 +0900",
		},
		{
			in:        "+++ sample.old.txt",
			filename:  "sample.old.txt",
			timestamp: "",
		},
	}
	for _, tt := range tests {
		gotf, gott := parseFileHeader(tt.in)
		if gotf != tt.filename || gott != tt.timestamp {
			t.Errorf("parseFileHeader(%v) = (%v, %v), want (%v, %v)", tt.in, gotf, gott, tt.filename, tt.timestamp)
		}
	}
}

func TestParseExtenedHeader(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{
			in: `diff --git a/sample.txt b/sample.txt
index a949a96..769bdae 100644
--- a/sample.old.txt
+++ b/sample.new.txt
@@ -1,3 +1,4 @@
`,
			want: []string{"diff --git a/sample.txt b/sample.txt", "index a949a96..769bdae 100644"},
		},
		{
			in: `diff --git a/sample.txt b/sample.txt
deleted file mode 100644
index e69de29..0000000
`,
			want: []string{"diff --git a/sample.txt b/sample.txt", "deleted file mode 100644", "index e69de29..0000000"},
		},
		{
			in: `diff --git a/sample.txt b/sample.txt
new file mode 100644
index 0000000..e69de29
diff --git a/sample2.txt b/sample2.txt
new file mode 100644
index 0000000..ee946eb
`,
			want: []string{"diff --git a/sample.txt b/sample.txt", "new file mode 100644", "index 0000000..e69de29"},
		},
		{
			in: `--- a/sample.old.txt
+++ b/sample.new.txt
@@ -1,3 +1,4 @@
`,
			want: nil,
		},
	}
	for _, tt := range tests {
		got := parseExtenedHeader(bufio.NewReader(strings.NewReader(tt.in)))
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("in:\n%v\ngot:\n%v\nwant:\n%v", tt.in, strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
		}
	}
}

func TestHunkParser_Parse(t *testing.T) {
	tests := []struct {
		in       string
		lnumdiff int
		want     *Hunk
	}{
		{
			in: `@@ -1,3 +1,4 @@ optional section heading
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
`,
			want: &Hunk{
				StartLineOld: 1, LineLengthOld: 3, StartLineNew: 1, LineLengthNew: 4,
				Section: "optional section heading",
				Lines: []*Line{
					{Type: 0, Content: "unchanged, contextual line", LnumDiff: 1, LnumOld: 1, LnumNew: 1},
					{Type: 1, Content: "deleted line", LnumDiff: 2, LnumOld: 2, LnumNew: 0},
					{Type: 1, Content: "added line", LnumDiff: 3, LnumOld: 0, LnumNew: 2},
					{Type: 1, Content: "added line", LnumDiff: 4, LnumOld: 0, LnumNew: 3},
					{Type: 0, Content: "unchanged, contextual line", LnumDiff: 5, LnumOld: 3, LnumNew: 4},
				},
			},
		},
		{
			in: `@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
@@ -1,3 +1,4 @@
`,
			lnumdiff: 14,
			want: &Hunk{
				StartLineOld: 1, LineLengthOld: 3, StartLineNew: 1, LineLengthNew: 4,
				Section: "",
				Lines: []*Line{
					{Type: 0, Content: "unchanged, contextual line", LnumDiff: 15, LnumOld: 1, LnumNew: 1},
					{Type: 1, Content: "deleted line", LnumDiff: 16, LnumOld: 2, LnumNew: 0},
					{Type: 1, Content: "added line", LnumDiff: 17, LnumOld: 0, LnumNew: 2},
					{Type: 1, Content: "added line", LnumDiff: 18, LnumOld: 0, LnumNew: 3},
					{Type: 0, Content: "unchanged, contextual line", LnumDiff: 19, LnumOld: 3, LnumNew: 4},
				},
			},
		},
	}
	for _, tt := range tests {
		got, err := (&hunkParser{r: bufio.NewReader(strings.NewReader(tt.in)), lnumdiff: tt.lnumdiff}).Parse()
		if err != nil {
			t.Errorf("hunkParser.Parse(%v) got an unexpected err %v", tt.in, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("hunkParser.Parse(%v) = \n%#v\n, want \n%#v", tt.in, got, tt.want)
			t.Logf("got lines:")
			for _, l := range got.Lines {
				t.Logf("%#v", l)
			}
			t.Logf("want lines:")
			for _, l := range tt.want.Lines {
				t.Logf("%#v", l)
			}
		}
	}
}

func TestParseHunkRange(t *testing.T) {
	tests := []struct {
		in   string
		want *hunkrange
	}{
		{
			in:   "@@ -1,3 +1,4 @@",
			want: &hunkrange{lold: 1, sold: 3, lnew: 1, snew: 4},
		},
		{
			in:   "@@ -1 +1 @@",
			want: &hunkrange{lold: 1, sold: 1, lnew: 1, snew: 1},
		},
		{
			in:   "@@ -1,3 +1,4 @@ optional section",
			want: &hunkrange{lold: 1, sold: 3, lnew: 1, snew: 4, section: "optional section"},
		},
	}
	for _, tt := range tests {
		got, err := parseHunkRange(tt.in)
		if err != nil {
			t.Errorf("parseHunkRange(%v) got an unexpected err %v", tt.in, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseHunkRange(%v) = %#v, want %#v", tt.in, got, tt.want)
		}
	}
}

func TestParseLS(t *testing.T) {
	tests := []struct {
		in string
		l  int
		s  int
	}{
		{in: "1,3", l: 1, s: 3},
		{in: "14", l: 14, s: 1},
	}
	for _, tt := range tests {
		gotl, gots, err := parseLS(tt.in)
		if err != nil {
			t.Errorf("parseLS(%v) got an unexpected err %v", tt.in, err)
		}
		if gotl != tt.l || gots != tt.s {
			t.Errorf("parseLS(%v) = (%v, %v, _), want (%v, %v, _)", tt.in, gotl, gots, tt.l, tt.s)
		}
	}
}

func TestReadline(t *testing.T) {
	text := `line1
line2
line3`
	r := bufio.NewReader(strings.NewReader(text))
	{
		got, err := readline(r)
		if err != nil {
			t.Error(err)
		}
		if got != "line1" {
			t.Errorf("got %v, want line1", got)
		}
	}
	{
		got, err := readline(r)
		if err != nil {
			t.Error(err)
		}
		if got != "line2" {
			t.Errorf("got %v, want line2", got)
		}
	}
	{
		got, err := readline(r)
		if err != nil {
			t.Error(err)
		}
		if got != "line3" {
			t.Errorf("got %v, want line3", got)
		}
	}
	{
		if _, err := readline(r); err != io.EOF {
			t.Errorf("got err %v, want io.EOF", err)
		}
	}
}
