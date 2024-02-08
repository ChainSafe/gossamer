// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pubip

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jpillora/backoff"
)

// MaxTries is the maximum amount of tries to attempt to one service.
const MaxTries = 3

// APIURIs is the URIs of the services.
var APIURIs = []string{
	"https://api.ipify.org",
	"http://ipinfo.io/ip",
	"http://checkip.amazonaws.com",
	"http://whatismyip.akamai.com",
	"http://ipv4.text.wtfismyip.com",
}

// Timeout sets the time limit of collecting results from different services.
var Timeout = 2 * time.Second

// GetIPBy queries an API to retrieve a `net.IP` of this machine's public IP
// address.
func GetIPBy(dest string) (net.IP, error) {
	b := &backoff.Backoff{
		Jitter: true,
	}
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, dest, nil)
	if err != nil {
		return nil, err
	}

	for tries := 0; tries < MaxTries; tries++ {
		resp, err := client.Do(req)
		if err != nil {
			d := b.Duration()
			time.Sleep(d)
			continue
		}

		defer resp.Body.Close() //nolint

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(dest + " status code " + strconv.Itoa(resp.StatusCode) + ", body: " + string(body))
		}

		tb := strings.TrimSpace(string(body))
		ip := net.ParseIP(tb)
		if ip == nil {
			return nil, errors.New("IP address not valid: " + tb)
		}
		return ip, nil
	}

	return nil, errors.New("Failed to reach " + dest)
}

func detailErr(err error, errs []error) error {
	errStrs := []string{err.Error()}
	for _, e := range errs {
		errStrs = append(errStrs, e.Error())
	}
	j := strings.Join(errStrs, "\n")
	return errors.New(j)
}

func validate(rs []net.IP) (net.IP, error) {
	if rs == nil {
		return nil, fmt.Errorf("failed to get any result from %d APIs", len(APIURIs))
	}
	if len(rs) < 3 {
		return nil, fmt.Errorf("less than %d results from %d APIs", 3, len(APIURIs))
	}
	first := rs[0]
	for i := 1; i < len(rs); i++ {
		if !reflect.DeepEqual(first, rs[i]) {
			return nil, fmt.Errorf("results are not identical: %s", rs)
		}
	}
	return first, nil
}

func worker(d string, r chan<- net.IP, e chan<- error) {
	ip, err := GetIPBy(d)
	if err != nil {
		e <- err
		return
	}
	r <- ip
}

// Get queries several APIs to retrieve a `net.IP` of this machine's public IP
// address.
func Get() (net.IP, error) {
	var results []net.IP
	resultCh := make(chan net.IP, len(APIURIs))
	var errs []error
	errCh := make(chan error, len(APIURIs))

	for _, d := range APIURIs {
		go worker(d, resultCh, errCh)
	}
	for {
		select {
		case err := <-errCh:
			errs = append(errs, err)
		case r := <-resultCh:
			results = append(results, r)
		case <-time.After(Timeout):
			r, err := validate(results)
			if err != nil {
				return nil, detailErr(err, errs)
			}
			return r, nil
		}
	}
}
