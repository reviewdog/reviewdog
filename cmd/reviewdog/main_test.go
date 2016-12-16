package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRun_local(t *testing.T) {
	const (
		before = `line1
line2
line3
`
		after = `line1
line2 changed
line3
`
	)

	beforef, err := ioutil.TempFile("", "reviewdog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer beforef.Close()
	defer os.Remove(beforef.Name())
	afterf, err := ioutil.TempFile("", "reviewdog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer afterf.Close()
	defer os.Remove(afterf.Name())

	beforef.WriteString(before)
	afterf.WriteString(after)

	fname := afterf.Name()

	var (
		stdin = strings.Join([]string{
			fname + "(2,1): message1",
			fname + "(2): message2",
			fname + "(14,1): message3",
		}, "\n")
		want = fname + "(2,1): message1"
	)

	diffCmd := fmt.Sprintf("diff -u %s %s", beforef.Name(), afterf.Name())

	opt := &option{
		diffCmd:   diffCmd,
		efms:      strslice([]string{`%f(%l,%c): %m`}),
		diffStrip: 0,
	}

	stdout := new(bytes.Buffer)
	if err := run(strings.NewReader(stdin), stdout, opt); err != nil {
		t.Error(err)
	}

	if got := strings.Trim(stdout.String(), "\n"); got != want {
		t.Errorf("raw: got %v, want %v", got, want)
	}

}

func TestRun_project(t *testing.T) {
	t.Run("diff command is empty", func(t *testing.T) {
		opt := &option{
			conf: "reviewdog.yml",
		}
		stdout := new(bytes.Buffer)
		if err := run(nil, stdout, opt); err == nil {
			t.Error("want err, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("config not found", func(t *testing.T) {
		opt := &option{
			conf:    "reviewdog.notfound.yml",
			diffCmd: "echo ''",
		}
		stdout := new(bytes.Buffer)
		if err := run(nil, stdout, opt); err == nil {
			t.Error("want err, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		conffile, err := ioutil.TempFile("", "reviewdog-test")
		if err != nil {
			t.Fatal(err)
		}
		defer conffile.Close()
		defer os.Remove(conffile.Name())
		conffile.WriteString("invalid yaml")
		opt := &option{
			conf:    conffile.Name(),
			diffCmd: "echo ''",
		}
		stdout := new(bytes.Buffer)
		if err := run(nil, stdout, opt); err == nil {
			t.Error("want err, got nil")
		} else {
			t.Log(err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		conffile, err := ioutil.TempFile("", "reviewdog-test")
		if err != nil {
			t.Fatal(err)
		}
		defer conffile.Close()
		defer os.Remove(conffile.Name())
		conffile.WriteString("") // empty
		opt := &option{
			conf:    conffile.Name(),
			diffCmd: "echo ''",
		}
		stdout := new(bytes.Buffer)
		if err := run(nil, stdout, opt); err != nil {
			t.Errorf("got unexpected err: %v", err)
		}
	})
}

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
		os.Unsetenv(key)
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
		os.Unsetenv(key)
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

	os.Unsetenv("CI_PULL_REQUEST")
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
		os.Unsetenv(key)
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
		os.Unsetenv(key)
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

func TestRun_version(t *testing.T) {
	stdout := new(bytes.Buffer)
	if err := run(nil, stdout, &option{version: true}); err != nil {
		t.Error(err)
	}
	if got := strings.TrimRight(stdout.String(), "\n"); got != version {
		t.Errorf("version = %v, want %v", got, version)
	}
}
