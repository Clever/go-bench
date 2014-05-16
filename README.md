## Developing

go-bench is built and tested against Go 1.2.
Ensure this is the version of Go you're running with `go version`.
Make sure your GOPATH is set, e.g. `export GOPATH=~/go`.
Clone the repository to a location outside your GOPATH, and symlink it to `$GOPATH/src/github.com/Clever/go-bench`.
If you have done all of the above, then you should be able to run

```
make
```

## Usage
You can run go-bench using:

	go run bench.go

Or:

	go build bench.go
	./bench

The following command-line flags are supported:

flag | required? | description
:---: | :---: | :---:
--speed | no; default 1 | Sets multiplier for playback speed
--output | no; not written if not provided | Path to file to which JSON output should be written
--root | yes | URL root for requests

go-bench reads requests to playback from standard input in the following format:
	
	time,method,path,auth,extra

item | required? | description
:---: | :---: | :---:
time | yes | Time in ms after initialization to send request
method | yes | HTTP method to use for request
path | yes | Path for request
auth | no | Basic authentication header value (unencoded)
extra | no | Information about the request that will be written to the output file

If you need a simple server to test your usage of go-bench, you can start one using:

	go run start_server.go