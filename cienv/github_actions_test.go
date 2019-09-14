package cienv

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetBuildInfoFromGitHubActionEventPath(t *testing.T) {
	got, _, err := getBuildInfoFromGitHubActionEventPath("_testdata/github_event_pull_request.json")
	if err != nil {
		t.Fatal(err)
	}
	want := &BuildInfo{Owner: "reviewdog", Repo: "reviewdog", SHA: "cb23119096646023c05e14ea708b7f20cee906d5", PullRequest: 285, Branch: "go1.13"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("result has diff:\n%s", diff)
	}
}
