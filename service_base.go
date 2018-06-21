package reviewdog

import (
	"fmt"
	"os/exec"
	"strings"
)

// `path` to `position`(Lnum for new file) to comment `body`s
type postedcomments map[string]map[int][]string

// IsPosted returns true if a given comment has been posted in code review service already,
// otherwise returns false. It sees comments with same path, same position,
// and same body as same comments.
func (p postedcomments) IsPosted(c *Comment, lineNum int) bool {
	if _, ok := p[c.Path]; !ok {
		return false
	}
	bodys, ok := p[c.Path][lineNum]
	if !ok {
		return false
	}
	for _, body := range bodys {
		if body == commentBody(c) {
			return true
		}
	}
	return false
}

func (p postedcomments) AddPostedComment(path string, lineNum int, body string) {
	if _, ok := p[path]; !ok {
		p[path] = make(map[int][]string)
	}
	if _, ok := p[path][lineNum]; !ok {
		p[path][lineNum] = make([]string, 0)
	}
	p[path][lineNum] = append(p[path][lineNum], body)
}

const bodyPrefix = `<sub>reported by [reviewdog](https://github.com/haya14busa/reviewdog) :dog:</sub>`

func commentBody(c *Comment) string {
	tool := ""
	if c.ToolName != "" {
		tool = fmt.Sprintf("**[%s]** ", c.ToolName)
	}
	return tool + bodyPrefix + "\n" + c.Body
}

func gitRelWorkdir() (string, error) {
	b, err := exec.Command("git", "rev-parse", "--show-prefix").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'git rev-parse --show-prefix': %v", err)
	}
	return strings.Trim(string(b), "\n"), nil
}
