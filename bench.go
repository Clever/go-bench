package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type RequestEvent struct {
	Verb  string
	Path  string
	User  string
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

func timeRequest(rootURL string, event RequestEvent) RequestResult {
	url := fmt.Sprintf("%s%s", rootURL, event.Path)
	req, _ := http.NewRequest(event.Verb, url, nil)
	if event.User != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(event.User+":"))))
	}

	var connectEndTime, headersEndTime, contentEndTime int64
	startTime := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: createDialFunc(startTime, &connectEndTime),
		},
	}

	r, err := client.Do(req)
	headersEndTime = time.Now().Sub(startTime).Nanoseconds()
	if err != nil {
		log.Fatalf("request err %#v: %s", req, err)
	}
	var contentSize int64 = 0
	defer r.Body.Close()
	if err == nil {
		buf := new(bytes.Buffer)
		contentSize, _ = io.Copy(buf, r.Body)
		contentEndTime = time.Now().Sub(startTime).Nanoseconds()
	}
	return RequestResult{err, r.StatusCode, contentSize, (connectEndTime) / 1000000,
		(headersEndTime - connectEndTime) / 1000000, (contentEndTime - headersEndTime) / 1000000}

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

func addToStats(event RequestEvent, result RequestResult) {
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

	if result.ResponseCode/100 < 6 {
		responseCodes[result.ResponseCode/100]++
	} else {
		responseCodes[0]++
	}

	requestLine := fmt.Sprintf("%s %s [%s]\n", event.Verb, event.Path, event.Extra)
	if result.Error != nil {
		colorPrint(8, fmt.Sprintln("%sGot error:", requestLine, result.Error))
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
		go func() { addToStats(rec, timeRequest(rootURL, rec)); mutex.Lock(); count--; mutex.Unlock() }()
	}

	for count > 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}

var outputWriter *bufio.Writer
var outputFile *os.File

func main() {
	speed := flag.Float64("speed", 1, "Sets multiplier for playback speed.")
	output := flag.String("output", "", "Output file for results, in json format.")
	rooturl := flag.String("root", "", "URL root for requests")
	flag.Parse()

	if *rooturl == "" {
		panic("rooturl parameter is required")
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

	parseAndReplay(os.Stdin, *rooturl, *speed)
	fmt.Println("Done!\n")
	if *output != "" {
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
