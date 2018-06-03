package main

import (
	"context"
	"os"
	"testing"

	"golang.org/x/oauth2"
)

func TestNewDoghouseCli(t *testing.T) {
	if _, ok := newDoghouseCli(context.Background()).Client.Transport.(*oauth2.Transport); ok {
		t.Error("got oauth2 http client, want default client")
	}

	const tokenEnv = "REVIEWDOG_TOKEN"
	saveToken := os.Getenv(tokenEnv)
	defer func() {
		if saveToken != "" {
			os.Setenv(tokenEnv, saveToken)
		} else {
			os.Unsetenv(tokenEnv)
		}
	}()
	os.Setenv(tokenEnv, "xxx")

	if _, ok := newDoghouseCli(context.Background()).Client.Transport.(*oauth2.Transport); !ok {
		t.Error("w/ TOKEN: got unexpected http client, want oauth client")
	}
}
