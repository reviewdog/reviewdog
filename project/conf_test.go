package project

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestParseNullRunner(t *testing.T) {
	for _, value := range []string{"", "null", "~"} {
		t.Run(value, func(t *testing.T) {
			_, err := Parse([]byte("runner:\n  lint: " + value + "\n"))
			if err == nil || !strings.Contains(err.Error(), `runner "lint"`) {
				t.Fatalf("Parse() error = %v, want named runner error", err)
			}
		})
	}
}

func TestParse(t *testing.T) {
	const yml = `
# reviewdog.yml

runner:
  golint:
    cmd: golint ./...
    level: info
    errorformat:
      - "%f:%l:%c: %m"
  govet:
    cmd: go tool vet -all -shadowstrict .
    format: govet
    level: warning
  namekey:
    cmd: echo 'name'
    name: nameoverwritten
    format: checkstyle
    level: error
`

	want := &Config{
		Runner: map[string]*Runner{
			"golint": {
				Cmd:         "golint ./...",
				Errorformat: []string{`%f:%l:%c: %m`},
				Name:        "golint",
				Level:       "info",
			},
			"govet": {
				Cmd:    "go tool vet -all -shadowstrict .",
				Format: "govet",
				Name:   "govet",
				Level:  "warning",
			},
			"namekey": {
				Cmd:    "echo 'name'",
				Format: "checkstyle",
				Name:   "nameoverwritten",
				Level:  "error",
			},
		},
	}

	got, err := Parse([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Parse() diff: (-got +want)\n%s", diff)
	}

}
