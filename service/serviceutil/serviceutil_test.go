package serviceutil

import (
	"os"
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
	if wd, _ := GitRelWorkdir(); wd != subDir {
		t.Fatalf("gitRelWorkdir() = %q, want %q", wd, subDir)
	}
}
