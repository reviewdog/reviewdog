package githubutils

import (
	"fmt"

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
