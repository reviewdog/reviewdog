package githubutils

import (
	"fmt"
	"net/url"
	"os"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

// LinkedMarkdownDiagnostic returns Markdown string which contains a link to the
// location in the diagnostic and the diagnostic content itself.
func LinkedMarkdownDiagnostic(owner, repo, sha string, d *rdf.Diagnostic) string {
	path := d.GetLocation().GetPath()
	msg := d.GetMessage()
	if path == "" {
		return msg
	}
	loc := BasicLocationFormat(d)
	line := int(d.GetLocation().GetRange().GetStart().GetLine())
	link, err := PathLink(owner, repo, sha, path, line)
	if err != nil {
		return fmt.Sprintf("%s %s", loc, msg)
	}
	return fmt.Sprintf("[%s](%s) %s", loc, link, msg)
}

// PathLink build a link to GitHub path to given sha, file, and line.
func PathLink(owner, repo, sha, path string, line int) (string, error) {
	serverURL, err := githubServerURL()
	if err != nil {
		return "", err
	}

	if sha == "" {
		sha = "master"
	}
	fragment := ""
	if line > 0 {
		fragment = fmt.Sprintf("#L%d", line)
	}

	result := fmt.Sprintf("%s/%s/%s/blob/%s/%s%s",
		serverURL.String(), owner, repo, sha, path, fragment)

	return result, nil
}

// BasicLocationFormat format a diagnostic to %f|%l col %c| errorformat.
func BasicLocationFormat(d *rdf.Diagnostic) string {
	loc := d.GetLocation()
	out := loc.GetPath() + "|"
	lnum := int(loc.GetRange().GetStart().GetLine())
	col := int(loc.GetRange().GetStart().GetColumn())
	if lnum != 0 {
		out = fmt.Sprintf("%s%d", out, lnum)
		if col != 0 {
			out = fmt.Sprintf("%s col %d", out, col)
		}
	}
	return out + "|"
}

const defaultGitHubServerURL = "https://github.com"

func githubServerURL() (*url.URL, error) {
	// get GitHub server URL from GitHub Actions' default environment variable GITHUB_SERVER_URL
	// ref: https://docs.github.com/en/actions/reference/environment-variables#default-environment-variables
	if baseURL := os.Getenv("GITHUB_SERVER_URL"); baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("GitHub server URL from GITHUB_SERVER_URL is invalid: %v, %w", baseURL, err)
		}
		return u, nil
	}
	u, err := url.Parse(defaultGitHubServerURL)
	if err != nil {
		return nil, fmt.Errorf("GitHub server URL from reviewdog default is invalid: %v, %w", defaultGitHubServerURL, err)
	}
	return u, nil
}
