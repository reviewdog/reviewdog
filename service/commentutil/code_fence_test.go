package commentutil

import (
	"io"
	"strings"
	"testing"
)

func TestGetCodeFenceLength(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{
			in:   "",
			want: 3,
		},
		{
			in:   "`inline code`",
			want: 3,
		},
		{
			in:   "``foo`bar``",
			want: 3,
		},
		{
			in:   "func main() {\nprintln(\"Hello World\")\n}\n",
			want: 3,
		},
		{
			in:   "```\nLook! You can see my backticks.\n```\n",
			want: 4,
		},
		{
			in:   "```go\nfunc main() {\nprintln(\"Hello World\")\n}\n```",
			want: 4,
		},
		{
			in:   "```go\nfunc main() {\nprintln(\"Hello World\")\n}\n`````",
			want: 6,
		},
		{
			in:   "`````\n````\n```",
			want: 6,
		},
	}

	for _, tt := range tests {
		want := tt.want
		if got := GetCodeFenceLength(tt.in); got != want {
			t.Errorf("got unexpected length.\ngot:\n%d\nwant:\n%d", got, want)
		}
	}
}

// A version of strings.Builder without WriteByte
type Builder struct {
	strings.Builder
	io.ByteWriter // conflicts with and hides strings.Builder's WriteByte.
}

func TestWriteCodeFence(t *testing.T) {
	var buf Builder
	err := WriteCodeFence(&buf, 10)
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := "``````````"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteCodeFenceWriteByte(t *testing.T) {
	var buf strings.Builder
	err := WriteCodeFence(&buf, 10)
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := "``````````"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
