package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/reviewdog/reviewdog/diff"
)

func main() {
	files, err := filepath.Glob("testdata/*.diff")
	if err != nil {
		panic(err)
	}
	for _, fname := range files {
		f, err := os.Open(fname)
		if err != nil {
			panic(err)
		}
		difffiles, err := diff.ParseMultiFile(f)
		if err != nil {
			panic(err)
		}
		out, err := os.Create(fname + ".json")
		if err != nil {
			panic(err)
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(difffiles); err != nil {
			panic(err)
		}
		out.Close()
		f.Close()
	}
}
