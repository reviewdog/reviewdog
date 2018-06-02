package ciutil

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
)

var (
	muTravisIPAddrs sync.RWMutex
	travisIPAddrs   = map[string]bool{}

	// https://www.appveyor.com/docs/build-environment/#ip-addresses
	appveyorIPAddrs = map[string]bool{
		"74.205.54.20":    true,
		"104.197.110.30":  true,
		"104.197.145.181": true,
		"146.148.85.29":   true,
		"67.225.139.254":  true,
		"67.225.138.82":   true,
		"67.225.139.144":  true,
		"138.91.141.243":  true,
	}
)

// IsFromCI returns true if given request is from tructed CI provider.
func IsFromCI(r *http.Request) bool {
	return IsFromTravisCI(r) || IsFromAppveyor(r)
}

// IsFromTravisCI returns true if given request is from Travis CI.
// https://docs.travis-ci.com/user/ip-addresses/
func IsFromTravisCI(r *http.Request) bool {
	muTravisIPAddrs.RLock()
	defer muTravisIPAddrs.RUnlock()
	return travisIPAddrs[ipFromReq(r)]
}

// https://docs.travis-ci.com/user/ip-addresses/
func UpdateTravisCIIPAddrs(cli *http.Client) error {
	ips, err := ipAddrs("nat.travisci.net", cli)
	if err != nil {
		return err
	}
	muTravisIPAddrs.Lock()
	defer muTravisIPAddrs.Unlock()
	travisIPAddrs = map[string]bool{}
	for _, ip := range ips {
		travisIPAddrs[ip] = true
	}
	return nil
}

// IsFromAppveyor returns true if given request is from Appveyor.
// https://www.appveyor.com/docs/build-environment/#ip-addresses
func IsFromAppveyor(r *http.Request) bool {
	return appveyorIPAddrs[ipFromReq(r)]
}

func ipFromReq(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Cannot use "github.com/miekg/dns" in Google App Engine.
// Use dnsjson.com instead as workaround.
func ipAddrs(target string, cli *http.Client) ([]string, error) {
	url := fmt.Sprintf("https://dnsjson.com/%s/A.json", target)
	c := http.DefaultClient
	if cli != nil {
		c = cli
	}
	r, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var res struct {
		Results struct {
			Records []string `json:"records"`
		} `json:"results"`
	}

	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Results.Records) == 0 {
		return nil, fmt.Errorf("failed to get IP addresses of %s", target)
	}

	return res.Results.Records, nil
}
