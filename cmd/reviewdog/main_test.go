package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/haya14busa/reviewdog"
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
		reporter:  "local",
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
			conf:     "reviewdog.yml",
			reporter: "local",
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
			conf:     "reviewdog.notfound.yml",
			diffCmd:  "echo ''",
			reporter: "local",
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
			conf:     conffile.Name(),
			diffCmd:  "echo ''",
			reporter: "local",
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
			conf:     conffile.Name(),
			diffCmd:  "echo ''",
			reporter: "local",
		}
		stdout := new(bytes.Buffer)
		if err := run(nil, stdout, opt); err != nil {
			t.Errorf("got unexpected err: %v", err)
		}
	})

	t.Run("conffile allows to be prefixed with '.' and '.yaml' file extension", func(t *testing.T) {
		for _, n := range []string{".reviewdog.yml", "reviewdog.yaml"} {
			f, err := os.OpenFile(n, os.O_RDONLY|os.O_CREATE, 0666)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			defer os.Remove(n)
			if _, err := readConf(n); err != nil {
				t.Errorf("readConf(%q) got unexpected err: %v", n, err)
			}
		}
	})
}

func TestGetPullRequestInfoFromEnv_travis(t *testing.T) {
	envs := []string{
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

	_, isPR, err := getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "str")

	_, isPR, err = getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "1")

	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "invalid repo slug")

	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_REPO_SLUG", "haya14busa/reviewdog")

	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("TRAVIS_PULL_REQUEST_SHA", "sha")

	_, isPR, err = getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if !isPR {
		t.Errorf("isPR = %v, want true", isPR)
	}

	os.Setenv("TRAVIS_PULL_REQUEST", "false")

	_, isPR, err = getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if isPR {
		t.Errorf("isPR = %v, want false", isPR)
	}
}

func TestGetPullRequestInfoFromEnv_circleci(t *testing.T) {
	envs := []string{
		"CIRCLE_PR_NUMBER",
		"CIRCLE_PROJECT_USERNAME",
		"CIRCLE_PROJECT_REPONAME",
		"CIRCLE_SHA1",
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

	if _, isPR, err := getPullRequestInfoFromEnv(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("CIRCLE_PR_NUMBER", "1")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PROJECT_USERNAME", "haya14busa")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_PROJECT_REPONAME", "reviewdog")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CIRCLE_SHA1", "sha1")
	g, isPR, err := getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &PullRequestInfo{
		owner: "haya14busa",
		repo:  "reviewdog",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetPullRequestInfoFromEnv_droneio(t *testing.T) {
	envs := []string{
		"DRONE_PULL_REQUEST",
		"DRONE_REPO",
		"DRONE_REPO_OWNER",
		"DRONE_REPO_NAME",
		"DRONE_COMMIT",
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

	if _, isPR, err := getPullRequestInfoFromEnv(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("DRONE_PULL_REQUEST", "1")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	// Drone <= 0.4 without valid repo
	os.Setenv("DRONE_REPO", "invalid")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
	os.Unsetenv("DRONE_REPO")

	// Drone > 0.4 without DRONE_REPO_NAME
	os.Setenv("DRONE_REPO_OWNER", "haya14busa")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}
	os.Unsetenv("DRONE_REPO_OWNER")

	// Drone > 0.4 without DRONE_REPO_OWNER
	os.Setenv("DRONE_REPO_NAME", "reviewdog")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	// Drone > 0.4 have valid variables
	os.Setenv("DRONE_REPO_NAME", "reviewdog")
	os.Setenv("DRONE_REPO_OWNER", "haya14busa")

	os.Setenv("DRONE_COMMIT", "sha1")
	g, isPR, err := getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &PullRequestInfo{
		owner: "haya14busa",
		repo:  "reviewdog",
		pr:    1,
		sha:   "sha1",
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("got: %#v, want: %#v", g, want)
	}
}

func TestGetPullRequestInfoFromEnv_common(t *testing.T) {
	envs := []string{
		"CI_PULL_REQUEST",
		"CI_COMMIT",
		"CI_REPO_OWNER",
		"CI_REPO_NAME",
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

	if _, isPR, err := getPullRequestInfoFromEnv(); isPR {
		t.Errorf("should be non pull-request build. error: %v", err)
	}

	os.Setenv("CI_PULL_REQUEST", "1")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_REPO_OWNER", "haya14busa")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_REPO_NAME", "reviewdog")
	if _, _, err := getPullRequestInfoFromEnv(); err == nil {
		t.Error("error expected but got nil")
	} else {
		t.Log(err)
	}

	os.Setenv("CI_COMMIT", "sha1")
	g, isPR, err := getPullRequestInfoFromEnv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !isPR {
		t.Error("should be pull request build")
	}
	want := &PullRequestInfo{
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
	if got := strings.TrimRight(stdout.String(), "\n"); got != reviewdog.Version {
		t.Errorf("version = %v, want %v", got, reviewdog.Version)
	}
}
