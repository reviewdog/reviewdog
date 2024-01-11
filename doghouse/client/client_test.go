package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/reviewdog/reviewdog/doghouse"
)

func TestDogHouseClient_Check(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected access: %v %v", r.Method, r.URL)
		}
		w.Write([]byte(`{"report_url": "http://report_url"}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := New(nil)
	cli.BaseURL, _ = url.Parse(ts.URL)

	req := &doghouse.CheckRequest{}
	resp, err := cli.Check(context.Background(), req, true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ReportURL != "http://report_url" {
		t.Errorf("got unexpected response: %v", resp)
	}
}

func TestDogHouseClient_Check_failure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`bad request!`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	cli := New(nil)
	cli.BaseURL, _ = url.Parse(ts.URL)

	req := &doghouse.CheckRequest{}
	_, err := cli.Check(context.Background(), req, true)
	if err == nil {
		t.Error("got no error, but want bad request error")
	}
}
