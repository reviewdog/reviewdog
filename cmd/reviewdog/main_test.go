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
		"REVIEWDOG_GITHUB_API_TOKEN",
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

	if err := run(nil, nil, &option{f: "golint", ci: "ciname"}); err != nil {
		t.Errorf("got an unexpected error: %v", err)
	}

	os.Setenv("REVIEWDOG_GITHUB_API_TOKEN", "<REVIEWDOG_GITHUB_API_TOKEN>")

	if err := run(nil, nil, &option{f: "golint", ci: "unsupported ci"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	if err := run(nil, nil, &option{f: "golint", ci: "travis"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "str")

	if err := run(nil, nil, &option{f: "golint", ci: "travis"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "1")

	if err := run(nil, nil, &option{f: "golint", ci: "travis"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "invalid repo slug")

	if err := run(nil, nil, &option{f: "golint", ci: "travis"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "haya14busa/reviewdog")

	if err := run(nil, nil, &option{f: "golint", ci: "travis"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST_SHA", "sha")

	r := strings.NewReader("compiler result")

	if err := run(r, new(bytes.Buffer), &option{diffCmd: "git diff", ci: "travis"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	buf := new(bytes.Buffer)
	os.Setenv("TRAVIS_PULL_REQUEST", "false")
	if err := run(r, buf, &option{f: "golint", ci: "travis"}); err != nil {
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
		"REVIEWDOG_GITHUB_API_TOKEN",
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

	os.Setenv("CIRCLE_PROJECT_REPONAME", "reviewdog")
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
		repo:  "reviewdog",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}

	os.Setenv("REVIEWDOG_GITHUB_API_TOKEN", "<REVIEWDOG_GITHUB_API_TOKEN>")
	if err := run(strings.NewReader("compiler result"), new(bytes.Buffer), &option{f: "golint", ci: "circle-ci"}); err == nil {
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
		"REVIEWDOG_GITHUB_API_TOKEN",
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

	os.Setenv("DRONE_REPO", "haya14busa/reviewdog")
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
		repo:  "reviewdog",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}

	os.Setenv("REVIEWDOG_GITHUB_API_TOKEN", "<REVIEWDOG_GITHUB_API_TOKEN>")
	if err := run(strings.NewReader("compiler result"), new(bytes.Buffer), &option{f: "golint", ci: "droneio"}); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
}

func TestCommonci(t *testing.T) {
	envs := []string{
		"CI_PULL_REQUEST",
		"CI_COMMIT",
		"CI_REPO_OWNER",
		"CI_REPO_NAME",
		"REVIEWDOG_GITHUB_API_TOKEN",
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

	if _, isPR, err := commonci(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("CI_PULL_REQUEST", "invalid")
	if _, _, err := commonci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_PULL_REQUEST", "1")
	if _, _, err := commonci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_REPO_OWNER", "haya14busa")
	if _, _, err := commonci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_REPO_NAME", "reviewdog")
	if _, _, err := commonci(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_COMMIT", "sha1")
	g, isPR, err := commonci()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &GitHubPR{
		owner: "haya14busa",
		repo:  "reviewdog",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}

}
