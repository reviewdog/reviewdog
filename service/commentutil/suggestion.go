package commentutil

import (
	"errors"
	"fmt"

	"github.com/reviewdog/reviewdog/proto/rdf"
)

// BuildSuggestionText applies a suggestion's replacement text to the source lines it covers.
func BuildSuggestionText(sourceLines map[int]string, s *rdf.Suggestion) (string, error) {
	if len(sourceLines) == 0 {
		return "", errors.New("source lines are not available")
	}
	start := s.GetRange().GetStart()
	end := s.GetRange().GetEnd()
	startLineContent, err := getSourceLine(sourceLines, int(start.GetLine()))
	if err != nil {
		return "", err
	}
	endLineContent, err := getSourceLine(sourceLines, int(end.GetLine()))
	if err != nil {
		return "", err
	}

	return startLineContent[:max(start.GetColumn()-1, 0)] + s.GetText() + endLineContent[max(end.GetColumn()-1, 0):], nil
}

func getSourceLine(sourceLines map[int]string, line int) (string, error) {
	lineContent, ok := sourceLines[line]
	if !ok {
		return "", fmt.Errorf("source line (L=%d) is not available for this suggestion", line)
	}
	return lineContent, nil
}
