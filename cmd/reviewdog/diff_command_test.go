package main

import "testing"

func TestDiffServiceSingleGitCommand(t *testing.T) {
	if _, err := diffService("git", 1); err != nil {
		t.Fatalf("diffService() error = %v", err)
	}
}
