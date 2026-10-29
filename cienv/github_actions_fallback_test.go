package cienv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetBuildInfoGitHubRepositoryFallback(t *testing.T) {
	tests := []struct {
		name, event, repository string
		want                    BuildInfo
		wantPR                  bool
	}{
		{name: "sparse scheduled event", event: `{"schedule":"0 0 * * *"}`, repository: "env-owner/env-repo", want: BuildInfo{Owner: "env-owner", Repo: "env-repo", SHA: "env-sha"}},
		{name: "missing owner", event: `{"repository":{"name":"payload-repo"}}`, repository: "env-owner/env-repo", want: BuildInfo{Owner: "env-owner", Repo: "payload-repo", SHA: "env-sha"}},
		{name: "missing repository name", event: `{"repository":{"owner":{"login":"payload-owner"}}}`, repository: "env-owner/env-repo", want: BuildInfo{Owner: "payload-owner", Repo: "env-repo", SHA: "env-sha"}},
		{name: "payload takes precedence", event: `{"repository":{"owner":{"login":"payload-owner"},"name":"payload-repo"},"head_commit":{"id":"event-sha"}}`, repository: "env-owner/env-repo", want: BuildInfo{Owner: "payload-owner", Repo: "payload-repo", SHA: "event-sha"}},
		{name: "pull request head is preserved", event: `{"pull_request":{"number":42,"head":{"sha":"pr-sha","ref":"feature"}}}`, repository: "env-owner/env-repo", want: BuildInfo{Owner: "env-owner", Repo: "env-repo", SHA: "pr-sha", PullRequest: 42, Branch: "feature"}, wantPR: true},
		{name: "check suite head is preserved", event: `{"check_suite":{"pull_requests":[{"number":43,"head":{"sha":"suite-sha","ref":"rerun"}}]}}`, repository: "env-owner/env-repo", want: BuildInfo{Owner: "env-owner", Repo: "env-repo", SHA: "suite-sha", PullRequest: 43, Branch: "rerun"}, wantPR: true},
		{name: "complete payload needs no environment repository", event: `{"repository":{"owner":{"login":"payload-owner"},"name":"payload-repo"}}`, want: BuildInfo{Owner: "payload-owner", Repo: "payload-repo", SHA: "env-sha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "event.json")
			if err := os.WriteFile(file, []byte(tt.event), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GITHUB_ACTIONS", "true")
			t.Setenv("GITHUB_EVENT_NAME", "schedule")
			t.Setenv("GITHUB_EVENT_PATH", file)
			t.Setenv("GITHUB_REPOSITORY", tt.repository)
			t.Setenv("GITHUB_SHA", "env-sha")
			got, isPR, err := GetBuildInfo()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(&tt.want, got); diff != "" {
				t.Errorf("build info (-want +got):\n%s", diff)
			}
			if isPR != tt.wantPR {
				t.Errorf("isPR=%t,want %t", isPR, tt.wantPR)
			}
		})
	}
}
