package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type RequestEvent struct {
	Verb string
	Path string
	User string
	Time int
}

type RequestEventReader struct {
	r *csv.Reader
}

func newRequestEventReader(r io.Reader) *RequestEventReader {
	reader := &RequestEventReader{r: csv.NewReader(r)}
	reader.r.TrailingComma = true
	return reader
}

func (r *RequestEventReader) Read() (event RequestEvent, err error) {
	line, err := r.r.Read()
	if err != nil {
		return RequestEvent{}, err
	}

	time, err := strconv.Atoi(line[0])
	if err != nil {
		return RequestEvent{}, err
	}
	return RequestEvent{line[1], line[2], line[3], time}, nil
}

func createDialFunc(startTime time.Time, endTimeResult *int64) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		conn, err := net.Dial(network, addr)
		*endTimeResult = time.Now().Sub(startTime).Nanoseconds()
		return conn, err
	}
}

func timeRequest(verb, url string) {
	req, _ := http.NewRequest(verb, url, nil)

	var connectEndTime, headersEndTime, contentEndTime int64
	startTime := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: createDialFunc(startTime, &connectEndTime),
		},
	}

	r, err := client.Do(req)
	headersEndTime = time.Now().Sub(startTime).Nanoseconds()

	defer r.Body.Close()
	if err != nil {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println("Status code: ", r.StatusCode)
		buf := new(bytes.Buffer)
		_, _ = io.Copy(buf, r.Body)
		contentEndTime = time.Now().Sub(startTime).Nanoseconds()
	}

	fmt.Println("Connect time:", (connectEndTime)/1000000, "ms")
	fmt.Println("Header receive time:", (headersEndTime-connectEndTime)/1000000, "ms")
	fmt.Println("Content receive time:", (contentEndTime-headersEndTime)/1000000, "ms")
}

func replayRequestEvent(rootURL string, event RequestEvent) {
	url := fmt.Sprintf("%s%s", rootURL, event.Path)
	fmt.Println(event.Verb, url)
	req, _ := http.NewRequest(event.Verb, url, nil)
	if event.User != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(event.User+":"))))
	}

	client := &http.Client{}
	r, err := client.Do(req)
	if err != nil {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println("Status code: ", r.StatusCode)
		r.Body.Close()
	}
}

func parseAndReplay(r io.Reader, rootURL string, speed float64) {
	var startTime time.Time
	in := newRequestEventReader(r)

	var mutex sync.Mutex
	count := 0
	for {
		rec, err := in.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		if startTime.IsZero() {
			startTime = time.Now()
		}

		for int(float64(time.Now().Sub(startTime)/time.Millisecond)*speed) < rec.Time {
			time.Sleep(time.Duration(100) * time.Millisecond)
		}
		mutex.Lock()
		count++
		mutex.Unlock()
		go func() { replayRequestEvent(rootURL, rec); mutex.Lock(); count--; mutex.Unlock() }()
	}

	for count > 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}

func main() {
	speed := flag.Float64("speed", 1, "Sets multiplier for playback speed.")
	flag.Parse()

	if flag.Arg(0) == "" {
		fmt.Println("Must specify a base URL.")
		os.Exit(1)
	}
	fmt.Println("Starting playback...")

	parseAndReplay(os.Stdin, flag.Arg(0), *speed)
}
