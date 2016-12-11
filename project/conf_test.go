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
    errorformat:
      - "%f:%l:%c: %m"
  govet:
    cmd: go tool vet -all -shadowstrict .
    format: govet
  namekey:
    cmd: echo 'name'
    name: nameoverwritten
    format: checkstyle
`

	want := &Config{
		Runner: map[string]*Runner{
			"golint": &Runner{
				Cmd:         "golint ./...",
				Errorformat: []string{`%f:%l:%c: %m`},
				Name:        "golint",
			},
			"govet": &Runner{
				Cmd:    "go tool vet -all -shadowstrict .",
				Format: "govet",
				Name:   "govet",
			},
			"namekey": &Runner{
				Cmd:    "echo 'name'",
				Format: "checkstyle",
				Name:   "nameoverwritten",
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
