package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRun_travis(t *testing.T) {
	envs := []string{
		"WATCHDOGS_GITHUB_API_TOKEN",
		"TRAVIS_PULL_REQUEST",
		"TRAVIS_REPO_SLUG",
		"TRAVIS_PULL_REQUEST_SHA",
	}
	// save and clean
	saveEnvs := make(map[string]string)
	for _, key := range envs {
		saveEnvs[key] = os.Getenv(key)
		os.Setenv(key, "")
	}
	// restore
	defer func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}()

	if err := run(nil, nil, "", 0, nil, "ciname"); err != nil {
		t.Errorf("got an unexpected error: %v", err)
	}

	os.Setenv("WATCHDOGS_GITHUB_API_TOKEN", "<WATCHDOGS_GITHUB_API_TOKEN>")

	if err := run(nil, nil, "", 0, nil, "unsupported ci"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	if err := run(nil, nil, "", 0, nil, "travis"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "str")

	if err := run(nil, nil, "", 0, nil, "travis"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "1")

	if err := run(nil, nil, "", 0, nil, "travis"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "invalid repo slug")

	if err := run(nil, nil, "", 0, nil, "travis"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "haya14busa/watchdogs")

	if err := run(nil, nil, "", 0, nil, "travis"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST_SHA", "sha")

	r := strings.NewReader("compiler result")

	if err := run(r, new(bytes.Buffer), "git diff", 0, nil, "travis"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	buf := new(bytes.Buffer)
	os.Setenv("TRAVIS_PULL_REQUEST", "false")
	if err := run(r, buf, "", 0, nil, "travis"); err != nil {
		t.Error(err)
	} else {
		t.Log(buf.String())
	}

}

func TestCircleci(t *testing.T) {
	envs := []string{
		"CI_PULL_REQUEST",
		"CIRCLE_PR_NUMBER",
		"CIRCLE_PROJECT_USERNAME",
		"CIRCLE_PROJECT_REPONAME",
		"CIRCLE_SHA1",
		"WATCHDOGS_GITHUB_API_TOKEN",
	}
	// save and clean
	saveEnvs := make(map[string]string)
	for _, key := range envs {
		saveEnvs[key] = os.Getenv(key)
		os.Setenv(key, "")
	}
	// restore
	defer func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}()

	if _, isPR, err := circleci(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("CI_PULL_REQUEST", "invalid")
	if _, _, err := circleci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_PULL_REQUEST", "")
	os.Setenv("CIRCLE_PR_NUMBER", "invalid")
	if _, _, err := circleci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PR_NUMBER", "1")
	if _, _, err := circleci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PROJECT_USERNAME", "haya14busa")
	if _, _, err := circleci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PROJECT_REPONAME", "watchdogs")
	if _, _, err := circleci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_SHA1", "sha1")
	g, isPR, err := circleci()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &GitHubPR{
		owner: "haya14busa",
		repo:  "watchdogs",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}

	os.Setenv("WATCHDOGS_GITHUB_API_TOKEN", "<WATCHDOGS_GITHUB_API_TOKEN>")
	if err := run(strings.NewReader("compiler result"), new(bytes.Buffer), "", 0, nil, "circle-ci"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
}

func TestDroneio(t *testing.T) {
	envs := []string{
		"DRONE_PULL_REQUEST",
		"DRONE_REPO",
		"DRONE_COMMIT",
		"WATCHDOGS_GITHUB_API_TOKEN",
	}
	// save and clean
	saveEnvs := make(map[string]string)
	for _, key := range envs {
		saveEnvs[key] = os.Getenv(key)
		os.Setenv(key, "")
	}
	// restore
	defer func() {
		for key, value := range saveEnvs {
			os.Setenv(key, value)
		}
	}()

	if _, isPR, err := droneio(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("DRONE_PULL_REQUEST", "invalid")
	if _, _, err := droneio(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("DRONE_PULL_REQUEST", "1")
	if _, _, err := droneio(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("DRONE_REPO", "invalid")
	if _, _, err := droneio(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("DRONE_REPO", "haya14busa/watchdogs")
	if _, _, err := droneio(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("DRONE_COMMIT", "sha1")
	g, isPR, err := droneio()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &GitHubPR{
		owner: "haya14busa",
		repo:  "watchdogs",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}

	os.Setenv("WATCHDOGS_GITHUB_API_TOKEN", "<WATCHDOGS_GITHUB_API_TOKEN>")
	if err := run(strings.NewReader("compiler result"), new(bytes.Buffer), "", 0, nil, "droneio"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
}
