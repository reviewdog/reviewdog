package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var fprint = flag.String("fprint", "", "fingerprint")
var toolName = flag.String("tool-name", "", "tool-name")

func main() {
	flag.Parse()
	if *fprint == "" || *toolName == "" {
		fmt.Println("Set both -fprint and -tool-name flags")
		os.Exit(1)
	}
	fmt.Println(serviceutil.EncodeMetaComment(*fprint, *toolName))
}
