package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/reviewdog/reviewdog/service/github"
)

var fprint = flag.String("fprint", "", "fingerprint")
var toolName = flag.String("tool-name", "", "tool-name")

func main() {
	flag.Parse()
	if fprint == nil || toolName == nil {
		fmt.Println("Set both -fprint and -tool-name flags")
		os.Exit(1)
	}
	fmt.Println(github.BuildMetaComment(*fprint, *toolName))
}
