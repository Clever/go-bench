package slowhttp

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultResponseCode   = 200
	DefaultHeaderSendTime = 100
	DefaultBodySendTime   = 200
)

// Parses a request like "GET /200/500/1000/1500 HTTP/1.1" into 200, 500, 1000, 1500
// Parse errors or missing values are silently ignored and replaced with defaults.
func parseRequest(request string) (responseCode, headerSendTime, bodySendTime int) {
	requestParts := strings.Fields(request)
	requestPathParts := strings.FieldsFunc(requestParts[1], func(r rune) bool { return r == '/' })

	responseCode = DefaultResponseCode
	headerSendTime = DefaultHeaderSendTime
	bodySendTime = DefaultBodySendTime

	if (len(requestPathParts)) >= 1 {
		v, err := strconv.Atoi(requestPathParts[0])
		if err == nil && v >= 0 && v < 1000 {
			responseCode = v
		}
	}

	if (len(requestPathParts)) >= 2 {
		v, err := strconv.Atoi(requestPathParts[1])
		if err == nil && v >= 0 {
			headerSendTime = v
		}
	}

	if (len(requestPathParts)) >= 3 {
		v, err := strconv.Atoi(requestPathParts[2])
		if err == nil && v >= 0 {
			bodySendTime = v
		}
	}

	return // Seriously, Go?
}

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.TrimRight(line, "\r\n"))
	responseCode, headerSendTime, bodySendTime := parseRequest(line)
	statusText := http.StatusText(responseCode)
	if statusText == "" {
		statusText = "Unknown"
	}

	line, err = reader.ReadString('\n')
	for ; line != "\r\n"; line, err = reader.ReadString('\n') {
		if err != nil {
			panic(err)
		}
		//fmt.Println("Got", strings.TrimRight(line, "\r\n"))
	}

	time.Sleep(time.Duration(headerSendTime) * time.Millisecond)
	writer.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", responseCode, statusText))
	writer.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	writer.WriteString("Cache-Control: no-store, no-cache, must-revalidate, max-age=0, post-check=0, pre-check=0")
	writer.WriteString("Pragma: no-cache")
	writer.WriteString("Connection-Type: close\r\n")
	writer.WriteString("\r\n")
	writer.Flush()

	time.Sleep(time.Duration(bodySendTime) * time.Millisecond)
	writer.WriteString(fmt.Sprintf("Response: %d (%s)\r\n", responseCode, statusText))
	writer.WriteString(fmt.Sprintf("Header Send Time: %d\r\n", headerSendTime))
	writer.WriteString(fmt.Sprintf("Body Send Time: %d\r\n", bodySendTime))
	writer.Flush()
	conn.Close()
}

func acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		go handleConnection(conn)
	}
}

func StartServer() (net.Listener, error) {
	listener, err := net.Listen("tcp", ":8653")
	fmt.Println("Starting server!")
	if err != nil {
		panic(err)
	}
	go acceptLoop(listener)
	return listener, nil
}
