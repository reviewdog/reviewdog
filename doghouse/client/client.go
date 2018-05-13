package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/haya14busa/reviewdog/doghouse"
)

const baseEndpoint = "https://review-dog.appspot.com"

type DogHouseClient struct {
	Client *http.Client
	// Base URL for API requests. Defaults is https://review-dog.appspot.com.
	BaseURL *url.URL
}

func New(client *http.Client) *DogHouseClient {
	dh := &DogHouseClient{Client: client}
	if dh.Client == nil {
		dh.Client = http.DefaultClient
	}
	dh.BaseURL, _ = url.Parse(baseEndpoint)
	return dh
}

func (c *DogHouseClient) Check(ctx context.Context, req *doghouse.CheckRequest) (*doghouse.CheckResponse, error) {
	url := c.BaseURL.String() + "/check"
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.WithContext(ctx)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Check request failed: %v", err)
	}
	defer httpResp.Body.Close()

	respb, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status=%v: %s", httpResp.StatusCode, respb)
	}

	var resp doghouse.CheckResponse
	if err := json.Unmarshal(respb, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: error=%v, resp=%s", err, respb)
	}
	return &resp, nil
}
