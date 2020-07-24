package ciutil

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

var (
	// Set Travis CI addrs by default though UpdateTravisCIIPAddrs will update it
	// with latest one.
	// $ dig +short nat.travisci.net | sort
	// # Updated on 2018-06-03.
	muTravisIPAddrs sync.RWMutex
	travisIPAddrs   = map[string]bool{
		"104.154.113.151": true,
		"104.154.120.187": true,
		"104.197.236.150": true,
		"146.148.51.141":  true,
		"146.148.58.237":  true,
		"147.75.192.163":  true,
		"207.254.16.35":   true,
		"207.254.16.36":   true,
		"207.254.16.37":   true,
		"207.254.16.38":   true,
		"207.254.16.39":   true,
		"34.233.56.198":   true,
		"34.234.4.53":     true,
		"35.184.226.236":  true,
		"35.184.48.144":   true,
		"35.184.96.71":    true,
		"35.188.184.134":  true,
		"35.188.1.99":     true,
		"35.188.73.34":    true,
		"35.192.136.167":  true,
		"35.192.187.174":  true,
		"35.192.19.50":    true,
		"35.192.217.12":   true,
		"35.192.85.2":     true,
		"35.193.203.142":  true,
		"35.193.211.2":    true,
		"35.193.7.13":     true,
		"35.202.145.110":  true,
		"35.202.68.136":   true,
		"35.202.78.106":   true,
		"35.224.112.202":  true,
		"35.226.126.204":  true,
		"52.3.55.28":      true,
		"52.45.185.117":   true,
		"52.45.220.64":    true,
		"52.54.31.11":     true,
		"52.54.40.118":    true,
		"54.208.31.17":    true,
	}

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

// IsFromCI returns true if given request is from trusted CI provider.
func IsFromCI(r *http.Request) bool {
	return IsFromTravisCI(r) || IsFromAppveyor(r)
}

// IsFromTravisCI returns true if given request is from Travis CI.
// https://docs.travis-ci.com/user/ip-addresses/
func IsFromTravisCI(r *http.Request) bool {
	muTravisIPAddrs.RLock()
	defer muTravisIPAddrs.RUnlock()
	return travisIPAddrs[IPFromReq(r)]
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
	return appveyorIPAddrs[IPFromReq(r)]
}

func IPFromReq(r *http.Request) string {
	if f := r.Header.Get("Forwarded"); f != "" {
		for _, kv := range strings.Split(f, ";") {
			if kvPair := strings.SplitN(kv, "=", 2); len(kvPair) == 2 &&
				strings.ToLower(strings.TrimSpace(kvPair[0])) == "for" {
				return strings.Trim(kvPair[1], ` "`)
			}
		}
	}

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
