package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type APIEvent struct {
	Verb       string
	RequestURL string
	User       string
	Time       int
}

type APIReader struct {
	r *csv.Reader
}

func NewAPIReader(r io.Reader) *APIReader {
	return &APIReader{r: csv.NewReader(r)}
}

func (r *APIReader) Read() (event APIEvent, err error) {
	line, err := r.r.Read()
	if err != nil {
		return APIEvent{}, err
	}

	time, err := strconv.Atoi(line[0])
	if err != nil {
		return APIEvent{}, err
	}
	return APIEvent{line[1], line[2], line[3], time}, nil
}

func replayAPIEvent(event APIEvent) {
	url := fmt.Sprintf("%s%s", "https://api-staging.ops.getclever.com", event.RequestURL)
	fmt.Println("URL: ", url)
	fmt.Println("VERB: ", event.Verb)
	req, _ := http.NewRequest(event.Verb, url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(event.User+":"))))

	client := &http.Client{}
	r, err := client.Do(req)
	if err != nil {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println("Status code: ", r.StatusCode)
		r.Body.Close()
	}
}

func CreateDialFunc(startTime time.Time, endTimeResult *int64) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		conn, err := net.Dial(network, addr)
		*endTimeResult = time.Now().Sub(startTime).Nanoseconds()
		return conn, err
	}
}

func TimeRequest(verb, url string) {
	req, _ := http.NewRequest(verb, url, nil)

	var connectEndTime, headersEndTime, contentEndTime int64
	startTime := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: CreateDialFunc(startTime, &connectEndTime),
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

func main() {
	fmt.Println("Starting log playback...")

	var startTime time.Time
	in := NewAPIReader(os.Stdin)
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

		for int(time.Now().Sub(startTime).Nanoseconds()/1000000) < rec.Time {
			time.Sleep(100000000)
		}
		fmt.Println(rec.Time, rec.Verb, rec.RequestURL)
	}
}
