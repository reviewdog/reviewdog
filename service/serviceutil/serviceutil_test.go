package serviceutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitRelWorkdir(t *testing.T) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	// Move to root dir.
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}

	wd, err := GitRelWorkdir()
	if err != nil {
		t.Fatal(err)
	}
	if wd != "" {
		t.Fatalf("gitRelWorkdir() = %q, want empty", wd)
	}
	subDir := "cmd/"
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	// GitRelWorkdir uses the OS-specific path separator.
	want := filepath.FromSlash(subDir)
	if wd, _ := GitRelWorkdir(); wd != want {
		t.Fatalf("gitRelWorkdir() = %q, want %q", wd, want)
	}
}
