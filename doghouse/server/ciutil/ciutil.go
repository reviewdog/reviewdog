package ciutil

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/miekg/dns"
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
		travisIPAddrs[ip.String()] = true
	}
	return nil
}

// IsFromAppveyor returns true if given request is from Appveyor.
// https://www.appveyor.com/docs/build-environment/#ip-addresses
func IsFromAppveyor(r *http.Request) bool {
	return appveyorIPAddrs[ipFromReq(r)]
}

func ipFromReq(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func ipAddrs(target string, cli *http.Client) ([]net.IP, error) {
	server := "8.8.8.8"
	c := dns.Client{Net: "tcp"}
	if cli != nil {
		c.HTTPClient = cli
	}
	m := dns.Msg{}
	m.SetQuestion(target+".", dns.TypeA)
	r, _, err := c.Exchange(&m, server+":53")
	if err != nil {
		return nil, err
	}
	if len(r.Answer) == 0 {
		return nil, fmt.Errorf("No results for %s", target)
	}
	addrs := make([]net.IP, 0, len(r.Answer))
	for _, ans := range r.Answer {
		if aRecord, ok := ans.(*dns.A); ok {
			addrs = append(addrs, aRecord.A)
		}
	}
	return addrs, nil
}
