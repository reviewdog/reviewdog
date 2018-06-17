package cienv

import (
	"os"
	"reflect"
	"testing"
)

func setupEnvs() (cleanup func()) {
	var cleanEnvs = []string{
		"CIRCLE_BRANCH",
		"CIRCLE_PROJECT_REPONAME",
		"CIRCLE_PROJECT_USERNAME",
		"CIRCLE_PR_NUMBER",
		"CIRCLE_SHA1",
		"CI_BRANCH",
		"CI_COMMIT",
		"CI_COMMIT_SHA",
		"CI_PROJECT_NAME",
		"CI_PROJECT_NAMESPACE",
		"CI_PULL_REQUEST",
		"CI_REPO_NAME",
		"CI_REPO_OWNER",
		"DRONE_COMMIT",
		"DRONE_COMMIT_BRANCH",
		"DRONE_PULL_REQUEST",
		"DRONE_REPO",
		"DRONE_REPO_NAME",
		"DRONE_REPO_OWNER",
		"TRAVIS_COMMIT",
		"TRAVIS_PULL_REQUEST",
		"TRAVIS_PULL_REQUEST_BRANCH",
		"TRAVIS_PULL_REQUEST_SHA",
		"TRAVIS_REPO_SLUG",
	}
	saveEnvs := make(map[string]string)
	for _, key := range cleanEnvs {
		saveEnvs[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	return func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}
}

func TestGetPullRequestInfo_travis(t *testing.T) {
	cleanup := setupEnvs()
	defer cleanup()

	os.Setenv("TRAVIS_REPO_SLUG", "invalid repo slug")

	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "haya14busa/reviewdog")

	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST_SHA", "sha")

	_, isPR, err := GetPullRequestInfo()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "str")

	_, isPR, err = GetPullRequestInfo()
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "1")

	if _, isPR, err = GetPullRequestInfo(); err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "false")

	_, isPR, err = GetPullRequestInfo()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}
}

func TestGetPullRequestInfo_circleci(t *testing.T) {
	cleanup := setupEnvs()
	defer cleanup()

	if _, isPR, err := GetPullRequestInfo(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("CIRCLE_PR_NUMBER", "1")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PROJECT_USERNAME", "haya14busa")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PROJECT_REPONAME", "reviewdog")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_SHA1", "sha1")
	g, isPR, err := GetPullRequestInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &PullRequestInfo{
		Owner:       "haya14busa",
		Repo:        "reviewdog",
		PullRequest: 1,
		SHA:         "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetPullRequestInfo_droneio(t *testing.T) {
	cleanup := setupEnvs()
	defer cleanup()

	if _, isPR, err := GetPullRequestInfo(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("DRONE_PULL_REQUEST", "1")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	// Drone <= 0.4 without valid repo
	os.Setenv("DRONE_REPO", "invalid")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
	os.Unsetenv("DRONE_REPO")

	// Drone > 0.4 without DRONE_REPO_NAME
	os.Setenv("DRONE_REPO_OWNER", "haya14busa")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
	os.Unsetenv("DRONE_REPO_OWNER")

	// Drone > 0.4 without DRONE_REPO_OWNER
	os.Setenv("DRONE_REPO_NAME", "reviewdog")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	// Drone > 0.4 have valid variables
	os.Setenv("DRONE_REPO_NAME", "reviewdog")
	os.Setenv("DRONE_REPO_OWNER", "haya14busa")

	os.Setenv("DRONE_COMMIT", "sha1")
	g, isPR, err := GetPullRequestInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &PullRequestInfo{
		Owner:       "haya14busa",
		Repo:        "reviewdog",
		PullRequest: 1,
		SHA:         "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetPullRequestInfo_common(t *testing.T) {
	cleanup := setupEnvs()
	defer cleanup()

	if _, isPR, err := GetPullRequestInfo(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("CI_PULL_REQUEST", "1")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_REPO_OWNER", "haya14busa")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_REPO_NAME", "reviewdog")
	if _, _, err := GetPullRequestInfo(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_COMMIT", "sha1")
	g, isPR, err := GetPullRequestInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &PullRequestInfo{
		Owner:       "haya14busa",
		Repo:        "reviewdog",
		PullRequest: 1,
		SHA:         "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}
