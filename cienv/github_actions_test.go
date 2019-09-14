package cienv

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetBuildInfoFromGitHubActionEventPath_prevent(t *testing.T) {
	got, _, err := getBuildInfoFromGitHubActionEventPath("_testdata/github_event_pull_request.json")
	if err != nil {
		t.Fatal(err)
	}
	want := &BuildInfo{Owner: "reviewdog", Repo: "reviewdog", SHA: "cb23119096646023c05e14ea708b7f20cee906d5", PullRequest: 285, Branch: "go1.13"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("result has diff:\n%s", diff)
	}
}

func TestGetBuildInfoFromGitHubActionEventPath_rerunevent(t *testing.T) {
	got, _, err := getBuildInfoFromGitHubActionEventPath("_testdata/github_event_rerun.json")
	if err != nil {
		t.Fatal(err)
	}
	want := &BuildInfo{Owner: "reviewdog", Repo: "reviewdog", SHA: "ba8f36cd3eb401e9de9ee5718e11d390fdbe4afa", PullRequest: 286, Branch: "github-actions-env"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("result has diff:\n%s", diff)
	}
}
