package main

import (
	slowhttp "github.com/Clever/go-bench/slowhttp"
	"os"
	"testing"
)

func assertNoError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestRequest_1(t *testing.T) {

}

func TestPlayback_1(t *testing.T) {
	p, err := os.Open("test/testdata1")
	assertNoError(err, t)

	l, err := slowhttp.StartServer()
	assertNoError(err, t)

	parseAndReplay(p, "http://127.0.0.1:8653", 1)
	l.Close()
}
