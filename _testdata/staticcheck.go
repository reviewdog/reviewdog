package testdata

import (
	"errors"
)

func unused() {
	if errors.New("abc") == errors.New("abc") {
		// test SA4000
	}
}
