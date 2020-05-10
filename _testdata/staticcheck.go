package testdata

import (
	"errors"
	"regexp"
)

func unused() {
	regexp.Compile(".+")

	if errors.New("abc") == errors.New("abc") {
		// test SA4000
	}
}
