package serviceutil

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"log"
	"strings"

	"github.com/reviewdog/reviewdog/proto/metacomment"
	"github.com/reviewdog/reviewdog/proto/rdf"

	"google.golang.org/protobuf/proto"
)

// DecodeMetaComment decodes a base64 encoded meta comment.
func DecodeMetaComment(metaBase64 string) (*metacomment.MetaComment, error) {
	b, err := base64.StdEncoding.DecodeString(metaBase64)
	if err != nil {
		return nil, err
	}
	meta := &metacomment.MetaComment{}
	if err := proto.Unmarshal(b, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// ExtractMetaComment extracts a meta comment from the given review comment body.
func ExtractMetaComment(body string) *metacomment.MetaComment {
	prefix := "<!-- __reviewdog__:"
	for _, line := range strings.Split(body, "\n") {
		if after, found := strings.CutPrefix(line, prefix); found {
			if metastring, foundSuffix := strings.CutSuffix(after, " -->"); foundSuffix {
				meta, err := DecodeMetaComment(metastring)
				if err != nil {
					log.Printf("failed to decode MetaComment: %v", err)
					continue
				}
				return meta
			}
		}
	}
	return nil
}

// EncodeMetaComment encodes meta comment as base64 string.
func EncodeMetaComment(fprint string, toolName string) string {
	b, _ := proto.Marshal(
		&metacomment.MetaComment{
			Fingerprint: fprint,
			SourceName:  toolName,
		},
	)
	return base64.StdEncoding.EncodeToString(b)
}

// BuildMetaComment builds a meta comment with the given fingerprint and tool name.
func BuildMetaComment(fprint string, toolName string) string {
	return fmt.Sprintf("<!-- __reviewdog__:%s -->", EncodeMetaComment(fprint, toolName))
}

// Fingerprint calculates a hash for the given diagnostic message.
func Fingerprint(d *rdf.Diagnostic) (string, error) {
	h := fnv.New64a()
	// Ideally, we should not use proto.Marshal since Proto Serialization Is Not
	// Canonical.
	// https://protobuf.dev/programming-guides/serialization-not-canonical/
	//
	// However, I left it as-is for now considering the same reviewdog binary
	// should re-calculate and compare fingerprint for almost all cases.
	data, err := proto.Marshal(d)
	if err != nil {
		return "", err
	}
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum64()), nil
}
