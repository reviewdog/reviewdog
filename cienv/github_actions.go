package cienv

import (
	"encoding/json"
	"errors"
	"os"
)

// https://docs.github.com/en/actions/reference/environment-variables#default-environment-variables
type GitHubEvent struct {
	PullRequest GitHubPullRequest `json:"pull_request"`
	Repository  struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`
	CheckSuite struct {
		After        string              `json:"after"`
		PullRequests []GitHubPullRequest `json:"pull_requests"`
	} `json:"check_suite"`
	HeadCommit struct {
		ID string `json:"id"`
	} `json:"head_commit"`
	ActionName string `json:"-"` // this is defined as env GITHUB_EVENT_NAME
}

type GitHubRepo struct {
	Owner struct {
		ID int64 `json:"id"`
	}
}

type GitHubPullRequest struct {
	Number int `json:"number"`
	Head   struct {
		Sha  string     `json:"sha"`
		Ref  string     `json:"ref"`
		Repo GitHubRepo `json:"repo"`
	} `json:"head"`
	Base struct {
		Repo GitHubRepo `json:"repo"`
	} `json:"base"`
}

// LoadGitHubEvent loads GitHubEvent if it's running in GitHub Actions.
func LoadGitHubEvent() (*GitHubEvent, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, errors.New("GITHUB_EVENT_PATH not found")
	}
	return loadGitHubEventFromPath(eventPath)
}

func loadGitHubEventFromPath(eventPath string) (*GitHubEvent, error) {
	f, err := os.Open(eventPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var event GitHubEvent
	if err := json.NewDecoder(f).Decode(&event); err != nil {
		return nil, err
	}
	event.ActionName = os.Getenv("GITHUB_EVENT_NAME")
	return &event, nil
}

func getBuildInfoFromGitHubAction() (*BuildInfo, bool, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, false, errors.New("GITHUB_EVENT_PATH not found")
	}
	return getBuildInfoFromGitHubActionEventPath(eventPath)
}
func getBuildInfoFromGitHubActionEventPath(eventPath string) (*BuildInfo, bool, error) {
	event, err := loadGitHubEventFromPath(eventPath)
	if err != nil {
		return nil, false, err
	}
	info := &BuildInfo{
		Owner:       event.Repository.Owner.Login,
		Repo:        event.Repository.Name,
		PullRequest: event.PullRequest.Number,
		Branch:      event.PullRequest.Head.Ref,
		SHA:         event.PullRequest.Head.Sha,
	}
	// For re-run check_suite event.
	if info.PullRequest == 0 && len(event.CheckSuite.PullRequests) > 0 {
		pr := event.CheckSuite.PullRequests[0]
		info.PullRequest = pr.Number
		info.Branch = pr.Head.Ref
		info.SHA = pr.Head.Sha
	}
	if info.SHA == "" {
		info.SHA = event.HeadCommit.ID
	}
	return info, info.PullRequest != 0, nil
}

// IsInGitHubAction returns true if reviewdog is running in GitHub Actions.
func IsInGitHubAction() bool {
	// https://docs.github.com/en/actions/configuring-and-managing-workflows/using-environment-variables#default-environment-variables
	return os.Getenv("GITHUB_ACTIONS") != ""
}

// HasReadOnlyPermissionGitHubToken returns true if reviewdog is running in GitHub
// Actions and running for PullRequests from forked repository with read-only token.
// https://docs.github.com/en/actions/reference/events-that-trigger-workflows#pull_request_target
func HasReadOnlyPermissionGitHubToken() bool {
	event, err := LoadGitHubEvent()
	if err != nil {
		return false
	}
	isForkedRepo := event.PullRequest.Head.Repo.Owner.ID != event.PullRequest.Base.Repo.Owner.ID
	return isForkedRepo && event.ActionName != "pull_request_target"
}
