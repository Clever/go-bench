package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type bodyResultData struct {
	RequestEvent
	Body interface{}
}

type RequestEvent struct {
	Verb  string
	Path  string
	Auth  string
	Time  int
	Extra string
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
	return RequestEvent{line[1], line[2], line[3], time, line[4]}, nil
}

func createDialFunc(startTime time.Time, endTimeResult *int64) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		conn, err := net.Dial(network, addr)
		*endTimeResult = time.Now().Sub(startTime).Nanoseconds()
		return conn, err
	}
}

type RequestResult struct {
	Error           error
	ResponseCode    int
	ContentSize     int64
	ConnectTime     int64
	HeaderSendTime  int64
	ContentSendTime int64
}

func eventToRequest(rootURL string, event RequestEvent) *http.Request {
	url := fmt.Sprintf("%s%s", rootURL, event.Path)
	req, err := http.NewRequest(event.Verb, url, nil)
	if err != nil {
		panic(err)
	}
	if event.Auth != "" {
		req.Header.Set("Authorization", event.Auth)
	}
	return req
}

func timeRequest(request *http.Request) (RequestResult, []byte) {
	var connectEndTime, headersEndTime, contentEndTime int64
	startTime := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: createDialFunc(startTime, &connectEndTime),
		},
	}

	r, err := client.Do(request)
	headersEndTime = time.Now().Sub(startTime).Nanoseconds()
	if err != nil {
		log.Fatalf("request err %#v: %s", request, err)
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("WARNING: Error while reading response body.", err.Error())
	}
	contentEndTime = time.Now().Sub(startTime).Nanoseconds()
	return RequestResult{err, r.StatusCode, int64(len(body)), (connectEndTime) / 1000000,
		(headersEndTime - connectEndTime) / 1000000, (contentEndTime - headersEndTime) / 1000000}, body

}

func colorPrint(color int, str string) {
	if color > 7 {
		fmt.Print("\x1b[1m")
		color -= 7
	}
	fmt.Printf("\x1b[3%dm%s\x1b[0m", color, str)
}

var responseCodes [6]int
var statsMutex sync.Mutex

func addToStats(event RequestEvent, result RequestResult, body []byte, bw io.Writer) {
	// To be implemented

	statsMutex.Lock()
	if outputWriter != nil {
		type ResultData struct {
			Verb string
			Path string
			RequestResult
			RequestTime int64
			Extra       string
		}
		data, err := json.Marshal(ResultData{event.Verb, event.Path, result, result.HeaderSendTime + result.ContentSendTime, event.Extra})
		if err != nil {
			panic(err)
		}
		outputWriter.Write(data)
		outputWriter.WriteRune('\n')
		outputWriter.Flush()
	}
	// Write the body data asynchronously.
	go func() {
		var bodymap interface{}
		if err := json.Unmarshal(body, &bodymap); err != nil {
			bodymap = string(body)
		}
		bodydata, err := json.Marshal(bodyResultData{event, bodymap})
		if err != nil {
			log.Println("WARNING: Error marshaling body result data.", err.Error())
		}
		bw.Write(append(bodydata, byte('\n')))
	}()

	if result.ResponseCode/100 < 6 {
		responseCodes[result.ResponseCode/100]++
	} else {
		responseCodes[0]++
	}

	requestLine := fmt.Sprintf("%s %s [%s]\n", event.Verb, event.Path, event.Extra)
	if result.Error != nil {
		colorPrint(8, fmt.Sprintf("%sGot error: %s\n", requestLine, result.Error))
	} else {
		resultLine := fmt.Sprintf("%sGot %d (%d bytes) in %d ms, %d ms, %d ms (%d ms)\n",
			requestLine, result.ResponseCode, result.ContentSize, result.ConnectTime,
			result.HeaderSendTime, result.ContentSendTime, result.HeaderSendTime+result.ContentSendTime)
		if result.ResponseCode < 300 {
			colorPrint(9, resultLine)
		} else if result.ResponseCode < 400 {
			colorPrint(11, resultLine)
		} else {
			colorPrint(8, resultLine)
		}
	}

	statsMutex.Unlock()
}

func parseAndReplay(r io.Reader, rootURL string, speed float64, bw io.Writer) {
	var startTime time.Time
	in := newRequestEventReader(r)

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
			time.Sleep(time.Duration(1) * time.Millisecond)
		}
		go func() {
			res, body := timeRequest(eventToRequest(rootURL, rec))
			addToStats(rec, res, body, bw)
		}()
	}

	// Sleep for a bit to wait for the last few requests to finish. This is a bit ad-hoc,
	// but is good enough for now.
	time.Sleep(time.Duration(100) * time.Millisecond)
}

var outputWriter *bufio.Writer
var outputFile *os.File

func main() {
	speed := flag.Float64("speed", 1, "Sets multiplier for playback speed.")
	output := flag.String("output", "", "Output file for results, in json format.")
	rooturl := flag.String("root", "", "URL root for requests")
	bodyoutput := flag.String("bodyoutput", "", "Output file for response bodies. Does not store body if empty.")
	flag.Parse()

	if *rooturl == "" {
		panic("root parameter is required")
	}

	if *output != "" {
		var err error
		outputFile, err = os.Create(*output)
		if err != nil {
			panic(err)
		}
		outputWriter = bufio.NewWriter(outputFile)
	}

	fmt.Println("Starting playback...")

	var bodyWriter io.Writer

	if len(*bodyoutput) == 0 {
		bodyWriter = ioutil.Discard
	} else {
		f, err := os.Create(*bodyoutput)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		bodyWriter = f
	}

	parseAndReplay(os.Stdin, *rooturl, *speed, bodyWriter)
	fmt.Println("Done!")
	if outputWriter != nil {
		outputWriter.Flush()
		outputFile.Close()
	}

	for i := 1; i < 6; i++ {
		if responseCodes[i] != 0 {
			fmt.Printf("%dxx count: %d\n", i, responseCodes[i])
		}
	}
	if responseCodes[0] != 0 {
		fmt.Printf("??? count: %d\n", responseCodes[0])
	}
}
