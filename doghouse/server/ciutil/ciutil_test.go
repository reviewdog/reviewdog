package ciutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsFromCI(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	const allowedIP = "67.225.139.254"
	r.RemoteAddr = allowedIP
	if !IsFromCI(r) {
		t.Errorf("IsFromCI(%q) = false, want true", allowedIP)
	}

	const notAllowedIP = "93.184.216.34"
	r.RemoteAddr = notAllowedIP
	if IsFromCI(r) {
		t.Errorf("IsFromCI(%q) = true, want false", notAllowedIP)
	}
}

func TestUpdateTravisCIIPAddrs(t *testing.T) {
	if err := UpdateTravisCIIPAddrs(nil); err != nil {
		t.Fatal(err)
	}
	if len(travisIPAddrs) == 0 {
		t.Fatal("travisIPAddrs is empty, want some ip addrs")
	}
	for addr := range travisIPAddrs {
		t.Log(addr)
	}
}

func TestIsFromTravisCI(t *testing.T) {
	if err := UpdateTravisCIIPAddrs(nil); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for addr := range travisIPAddrs {
		r.RemoteAddr = addr
		if !IsFromTravisCI(r) {
			t.Errorf("IsIsFromTravisCI(%q) = false, want true", r.RemoteAddr)
		}
	}

	const notAllowedIP = "93.184.216.34"
	r.RemoteAddr = notAllowedIP
	if IsFromTravisCI(r) {
		t.Errorf("IsFromTravisCI(%q) = true, want false", notAllowedIP)
	}
}

func TestIsFromAppveyor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	const allowedIP = "67.225.139.254"
	r.RemoteAddr = allowedIP
	if !IsFromAppveyor(r) {
		t.Errorf("IsFromAppveyor(%q) = false, want true", allowedIP)
	}

	const notAllowedIP = "93.184.216.34"
	r.RemoteAddr = notAllowedIP
	if IsFromAppveyor(r) {
		t.Errorf("IsFromAppveyor(%q) = true, want false", notAllowedIP)
	}
}
