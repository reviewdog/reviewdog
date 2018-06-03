// Package cienv provides utility for environment variable in CI services.
package cienv

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// PullRequestInfo represents required information about GitHub PullRequest.
type PullRequestInfo struct {
	Owner       string
	Repo        string
	PullRequest int
	SHA         string
}

// GetPullRequestInfo returns PullRequestInfo from environment variables.
//
// Supporrted CI services' documents:
// - Travis CI: https://docs.travis-ci.com/user/environment-variables/
// - Circle CI: https://circleci.com/docs/environment-variables/
// - Drone.io: http://docs.drone.io/environment-reference/
func GetPullRequestInfo() (prInfo *PullRequestInfo, isPR bool, err error) {
	pr := getPullRequestNum()
	if pr == 0 {
		return nil, false, nil
	}

	owner, repo := getOwnerAndRepoFromSlug([]string{
		"TRAVIS_REPO_SLUG",
		"DRONE_REPO", // drone<=0.4
	})
	if owner == "" {
		owner = getOneEnvValue([]string{
			"CI_REPO_OWNER", // common
			"CIRCLE_PROJECT_USERNAME",
			"DRONE_REPO_OWNER",
		})
	}
	if owner == "" {
		return nil, false, errors.New("cannot get repo owner from environment variable. Set CI_REPO_OWNER?")
	}

	if repo == "" {
		repo = getOneEnvValue([]string{
			"CI_REPO_NAME", // common
			"CIRCLE_PROJECT_REPONAME",
			"DRONE_REPO_NAME",
		})
	}

	if owner == "" {
		return nil, false, errors.New("cannot get repo name from environment variable. Set CI_REPO_NAME?")
	}

	sha := getOneEnvValue([]string{
		"CI_COMMIT", // common
		"TRAVIS_PULL_REQUEST_SHA",
		"CIRCLE_SHA1",
		"DRONE_COMMIT",
	})
	if sha == "" {
		return nil, false, errors.New("cannot get commit SHA from environment variable. Set CI_COMMIT?")
	}

	return &PullRequestInfo{
		Owner:       owner,
		Repo:        repo,
		PullRequest: pr,
		SHA:         sha,
	}, true, nil
}

func getPullRequestNum() int {
	envs := []string{
		// Common.
		"CI_PULL_REQUEST",
		// Travis CI.
		"TRAVIS_PULL_REQUEST",
		// Circle CI.
		"CIRCLE_PULL_REQUEST", // CircleCI 2.0
		"CIRCLE_PR_NUMBER",    // For Pull Request by a fork repository
		// drone.io.
		"DRONE_PULL_REQUEST",
	}
	// regexp.MustCompile() in func intentionally because this func is called
	// once for one run.
	re := regexp.MustCompile(`[1-9]\d*$`)
	for _, env := range envs {
		prm := re.FindString(os.Getenv(env))
		pr, _ := strconv.Atoi(prm)
		if pr != 0 {
			return pr
		}
	}
	return 0
}

func getOneEnvValue(envs []string) string {
	for _, env := range envs {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

func getOwnerAndRepoFromSlug(slugEnvs []string) (string, string) {
	repoSlug := getOneEnvValue(slugEnvs)
	ownerAndRepo := strings.SplitN(repoSlug, "/", 2)
	if len(ownerAndRepo) < 2 {
		return "", ""
	}
	return ownerAndRepo[0], ownerAndRepo[1]
}
