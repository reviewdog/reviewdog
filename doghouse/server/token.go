package server

import (
	"crypto/rand"
	"fmt"
	"io"
)

func GenerateRepositoryToken() string {
	return securerandom(8)
}

func securerandom(n int) string {
	b := make([]byte, n)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)
}
