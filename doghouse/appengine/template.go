//go:generate statik -src=./tmpl
package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/rakyll/statik/fs"

	_ "github.com/reviewdog/reviewdog/doghouse/appengine/statik"
)

var tmplFiles http.FileSystem

func mustParseTemplatesFiles(filenames ...string) *template.Template {
	t, err := parseTemplatesFiles(filenames...)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func parseTemplatesFiles(filenames ...string) (*template.Template, error) {
	var t *template.Template
	for _, filename := range filenames {
		if t == nil {
			t = template.New(filename)
		}
		text, err := fs.ReadFile(tmplFiles, filename)
		if err != nil {
			return nil, err
		}
		t, err = t.Parse(string(text))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

var (
	topTmpl    *template.Template
	ghTopTmpl  *template.Template
	ghRepoTmpl *template.Template
)

func initTemplates() {
	var err error
	tmplFiles, err = fs.New()
	if err != nil {
		log.Fatal(err)
	}

	topTmpl = mustParseTemplatesFiles(
		"/base.html",
		"/index.html",
	)

	ghTopTmpl = mustParseTemplatesFiles(
		"/gh/base.html",
		"/gh/header.html",
		"/gh/top.html",
	)

	ghRepoTmpl = mustParseTemplatesFiles(
		"/gh/base.html",
		"/gh/header.html",
		"/gh/repo.html",
	)
}
