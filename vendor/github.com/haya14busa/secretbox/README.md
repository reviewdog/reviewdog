[![LICENSE](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/haya14busa/secretbox?status.svg)](https://godoc.org/github.com/haya14busa/secretbox)

# secretbox

Package secretbox provides utility wrapper of https://godoc.org/golang.org/x/crypto/nacl/secretbox

## Library

[![GoDoc](https://godoc.org/github.com/haya14busa/secretbox?status.svg)](https://godoc.org/github.com/haya14busa/secretbox)


```go
import "github.com/haya14busa/secretbox"
```

```go
// You can generate key with the following command.
// ruby -rsecurerandom -e 'puts SecureRandom.hex(32)'
const key = "0f5297b6f0114171e9de547801b1e8bb929fe1d091e63c6377a392ec1baa3d0b"
s, err := NewFromHexKey(key)
if err != nil {
  panic(err)
}
plaintext := "vim vim vim"

// Encrypt
ciphertext, _ := s.Encrypt([]byte(plaintext))

// Decrypt
b, err := s.Decrypt(ciphertext)
if err != nil {
  panic(err)
}

fmt.Printf("%s", b)
// OUTPUT: vim vim vim
```

## CLI

```
go get -u github.com/haya14busa/secretbox/cmd/secretbox
```

```
$ export KEY=$(ruby -rsecurerandom -e 'puts SecureRandom.hex(32)')
$ echo 'vim or not vim, that is the question' | secretbox -key="${KEY}" > /tmp/ciphertext
$ secretbox -key="${KEY}" -d < /tmp/ciphertext
vim or not vim, that is the question
```
