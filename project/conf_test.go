package project

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

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
