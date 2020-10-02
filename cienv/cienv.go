// Package cienv provides utility for environment variable in CI services.
package cienv

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// BuildInfo represents build information about GitHub or GitLab project.
type BuildInfo struct {
	Owner string
	Repo  string
	SHA   string

	// Optional.
	PullRequest int // MergeRequest for GitLab.

	// Optional.
	Branch string

	// Gerrit related params
	GerritChangeID   string
	GerritRevisionID string
}

// GetBuildInfo returns BuildInfo from environment variables.
//
// Supported CI services' documents:
// - Travis CI: https://docs.travis-ci.com/user/environment-variables/
// - Circle CI: https://circleci.com/docs/environment-variables/
// - Drone.io: http://docs.drone.io/environment-reference/
// - GitLab CI: https://docs.gitlab.com/ee/ci/variables/#predefined-variables-environment-variables
// - GitLab CI doesn't export ID of Merge Request. https://gitlab.com/gitlab-org/gitlab-ce/issues/15280
func GetBuildInfo() (prInfo *BuildInfo, isPR bool, err error) {
	if IsInGitHubAction() {
		return getBuildInfoFromGitHubAction()
	}
	owner, repo := getOwnerAndRepoFromSlug([]string{
		"TRAVIS_REPO_SLUG",
		"DRONE_REPO", // drone<=0.4
		"BITBUCKET_REPO_FULL_NAME",
	})
	if owner == "" {
		owner = getOneEnvValue([]string{
			"CI_REPO_OWNER", // common
			"CIRCLE_PROJECT_USERNAME",
			"DRONE_REPO_OWNER",
			"CI_PROJECT_NAMESPACE", // GitLab CI
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
			"CI_PROJECT_NAME", // GitLab CI
		})
	}

	if repo == "" {
		return nil, false, errors.New("cannot get repo name from environment variable. Set CI_REPO_NAME?")
	}

	sha := getOneEnvValue([]string{
		"CI_COMMIT", // common
		"TRAVIS_PULL_REQUEST_SHA",
		"TRAVIS_COMMIT",
		"CIRCLE_SHA1",
		"DRONE_COMMIT",
		"CI_COMMIT_SHA", // GitLab CI
		"BITBUCKET_COMMIT",
	})
	if sha == "" {
		return nil, false, errors.New("cannot get commit SHA from environment variable. Set CI_COMMIT?")
	}

	branch := getOneEnvValue([]string{
		"CI_BRANCH", // common
		"TRAVIS_PULL_REQUEST_BRANCH",
		"CIRCLE_BRANCH",
		"DRONE_COMMIT_BRANCH",
		// present only if PR pipeline
		"BITBUCKET_PR_DESTINATION_BRANCH",
		"BITBUCKET_BRANCH",
	})

	pr := getPullRequestNum()

	return &BuildInfo{
		Owner:       owner,
		Repo:        repo,
		PullRequest: pr,
		SHA:         sha,
		Branch:      branch,
	}, pr != 0, nil
}

// GetGerritBuildInfo returns Gerrit specific build info
func GetGerritBuildInfo() (*BuildInfo, error) {
	changeID := os.Getenv("GERRIT_CHANGE_ID")
	if changeID == "" {
		return nil, errors.New("cannot get change id from environment variable. Set GERRIT_CHANGE_ID ?")
	}

	revisionID := os.Getenv("GERRIT_REVISION_ID")
	if revisionID == "" {
		return nil, errors.New("cannot get revision id from environment variable. Set GERRIT_REVISION_ID ?")
	}

	branch := os.Getenv("GERRIT_BRANCH")
	if branch == "" {
		return nil, errors.New("cannot get branch from environment variable. Set GERRIT_BRANCH ?")
	}

	return &BuildInfo{
		GerritChangeID:   changeID,
		GerritRevisionID: revisionID,
		Branch:           branch,
	}, nil
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
		// GitLab CI MergeTrains
		"CI_MERGE_REQUEST_IID",
		"BITBUCKET_PR_ID",
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
