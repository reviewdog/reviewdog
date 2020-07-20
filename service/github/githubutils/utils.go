package githubutils

import (
	"fmt"

	"github.com/reviewdog/reviewdog"
)

// LinkedMarkdownCheckResult returns Markdown string which contains a link to the
// location in the CheckResult and CheckResult content itself.
func LinkedMarkdownCheckResult(owner, repo, sha string, c *reviewdog.CheckResult) string {
	path := c.Diagnostic.GetLocation().GetPath()
	msg := c.Diagnostic.GetMessage()
	if path == "" {
		return msg
	}
	loc := BasicLocationFormat(c)
	line := int(c.Diagnostic.GetLocation().GetRange().GetStart().GetLine())
	link := PathLink(owner, repo, sha, path, line)
	return fmt.Sprintf("[%s](%s) %s", loc, link, msg)
}

// PathLink build a link to GitHub path to given sha, file, and line.
func PathLink(owner, repo, sha, path string, line int) string {
	if sha == "" {
		sha = "master"
	}
	fragment := ""
	if line > 0 {
		fragment = fmt.Sprintf("#L%d", line)
	}
	return fmt.Sprintf("http://github.com/%s/%s/blob/%s/%s%s",
		owner, repo, sha, path, fragment)
}

// BasicLocationFormat format check CheckResult to %f|%l col %c| errorformat.
func BasicLocationFormat(c *reviewdog.CheckResult) string {
	loc := c.Diagnostic.GetLocation()
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
