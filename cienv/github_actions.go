package cienv

import (
	"encoding/json"
	"errors"
	"os"
)

// https://help.github.com/en/articles/contexts-and-expression-syntax-for-github-actions#github-context
type GitHubContext struct {
	Sha       string `json:"sha"`
	HeadRef   string `json:"head_ref"`
	EventName string `json:"event_name"`
	Event     struct {
		Number      int `json:"number"`
		PullRequest struct {
			Head struct {
				Sha string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
		Repository struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			Name string `json:"name"`
		} `json:"repository"`
	} `json:"event"`
}

func getBuildInfoFromGitHubAction() (*BuildInfo, bool, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, false, errors.New("GITHUB_EVENT_PATH not found")
	}
	return getBuildInfoFromGitHubActionEventPath(eventPath)
}
func getBuildInfoFromGitHubActionEventPath(eventPath string) (*BuildInfo, bool, error) {
	f, err := os.Open(eventPath)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	var ghCtx GitHubContext
	if err := json.NewDecoder(f).Decode(&ghCtx); err != nil {
		return nil, false, err
	}
	info := &BuildInfo{
		Owner:       ghCtx.Event.Repository.Owner.Login,
		Repo:        ghCtx.Event.Repository.Name,
		PullRequest: ghCtx.Event.Number,
		Branch:      ghCtx.HeadRef,
	}
	if ghCtx.Event.PullRequest.Head.Sha != "" {
		info.SHA = ghCtx.Event.PullRequest.Head.Sha
	} else {
		info.SHA = ghCtx.Sha
	}
	return info, info.PullRequest != 0, nil
}

func IsInGitHubAction() bool {
	// https://help.github.com/en/articles/virtual-environments-for-github-actions#default-environment-variables
	return os.Getenv("GITHUB_ACTION") != ""
}
