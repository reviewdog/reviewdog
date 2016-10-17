package main

import (
	"bytes"
	"os"
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

	if err := run(nil, nil, "", 0, nil, "ciname"); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
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
