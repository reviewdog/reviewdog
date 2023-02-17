package cienv

import (
	"os"
	"reflect"
	"testing"
)

func TestGetBuildInfo_travis(t *testing.T) {
	t.Setenv("TRAVIS_REPO_SLUG", "invalid repo slug")

	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("TRAVIS_REPO_SLUG", "haya14busa/reviewdog")

	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("TRAVIS_PULL_REQUEST_SHA", "sha")

	_, isPR, err := GetBuildInfo()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}

	t.Setenv("TRAVIS_PULL_REQUEST", "str")

	_, isPR, err = GetBuildInfo()
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}

	t.Setenv("TRAVIS_PULL_REQUEST", "1")

	if _, isPR, err = GetBuildInfo(); err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}

	t.Setenv("TRAVIS_PULL_REQUEST", "false")

	_, isPR, err = GetBuildInfo()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}
}

func TestGetBuildInfo_circleci(t *testing.T) {
	if _, isPR, err := GetBuildInfo(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	t.Setenv("CIRCLE_PR_NUMBER", "1")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("CIRCLE_PROJECT_USERNAME", "haya14busa")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("CIRCLE_PROJECT_REPONAME", "reviewdog")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("CIRCLE_SHA1", "sha1")
	g, isPR, err := GetBuildInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &BuildInfo{
		Owner:       "haya14busa",
		Repo:        "reviewdog",
		PullRequest: 1,
		SHA:         "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetBuildInfo_droneio(t *testing.T) {
	if _, isPR, err := GetBuildInfo(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	t.Setenv("DRONE_PULL_REQUEST", "1")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	// Drone <= 0.4 without valid repo
	t.Setenv("DRONE_REPO", "invalid")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
	os.Unsetenv("DRONE_REPO")

	// Drone > 0.4 without DRONE_REPO_NAME
	t.Setenv("DRONE_REPO_OWNER", "haya14busa")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
	os.Unsetenv("DRONE_REPO_OWNER")

	// Drone > 0.4 without DRONE_REPO_OWNER
	t.Setenv("DRONE_REPO_NAME", "reviewdog")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	// Drone > 0.4 have valid variables
	t.Setenv("DRONE_REPO_NAME", "reviewdog")
	t.Setenv("DRONE_REPO_OWNER", "haya14busa")

	t.Setenv("DRONE_COMMIT", "sha1")
	g, isPR, err := GetBuildInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &BuildInfo{
		Owner:       "haya14busa",
		Repo:        "reviewdog",
		PullRequest: 1,
		SHA:         "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetBuildInfo_common(t *testing.T) {
	if _, isPR, err := GetBuildInfo(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	t.Setenv("CI_PULL_REQUEST", "1")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("CI_REPO_OWNER", "haya14busa")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("CI_REPO_NAME", "reviewdog")
	if _, _, err := GetBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("CI_COMMIT", "sha1")
	g, isPR, err := GetBuildInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &BuildInfo{
		Owner:       "haya14busa",
		Repo:        "reviewdog",
		PullRequest: 1,
		SHA:         "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetGerritBuildInfo(t *testing.T) {
	// without any environment variables
	if _, err := GetGerritBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("GERRIT_CHANGE_ID", "changedID1")
	if _, err := GetGerritBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("GERRIT_REVISION_ID", "revisionID1")
	if _, err := GetGerritBuildInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	t.Setenv("GERRIT_BRANCH", "master")
	if _, err := GetGerritBuildInfo(); err != nil {
		t.Error("nil expected but got err")
	}
}
