package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	slowhttp "github.com/clever/go-bench/slowhttp"
	"github.com/stretchr/testify/assert"
)

func assertNoError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestPlayback_1(t *testing.T) {
	p, err := os.Open("test/testdata1")
	assertNoError(err, t)

	l, err := slowhttp.StartServer()
	assertNoError(err, t)

	parseAndReplay(p, "http://127.0.0.1:8653", 1, ioutil.Discard)
	l.Close()
}

func TestEventToRequestWithEmptyAuthHeader(t *testing.T) {
	assertRequestMatchesExpected(t, "", nil)
}

func TestEventToRequestWithBasicAuthHeader(t *testing.T) {
	assertRequestMatchesExpected(t, "Basic XXXX", []string{"Basic XXXX"})
}

func TestEventToRequestWithBearerAuthHeader(t *testing.T) {
	assertRequestMatchesExpected(t, "Bearer YYYY", []string{"Bearer YYYY"})
}

func assertRequestMatchesExpected(t *testing.T, authParam string, expectedAuthHeader []string) {
	const rootURL = "http://127.0.0.1:8653"
	const verb = "GET"
	const path = "/test/path"
	const delay = 0
	const extra = ""

	requestEvent := RequestEvent{verb, path, authParam, delay, extra}
	httpRequest := eventToRequest(rootURL, requestEvent)
	assert.Equal(t, verb, httpRequest.Method, "Wrong HTTP method")
	host := strings.Replace(rootURL, "http://", "", -1)
	assert.Equal(t, host, httpRequest.Host, "Wrong root URL")
	assert.Equal(t, path, httpRequest.URL.Path, "Wrong request path")
	assert.Equal(t, expectedAuthHeader, httpRequest.Header["Authorization"], "Wrong authentication header")
}
