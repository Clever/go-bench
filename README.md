## Developing

```
go get github.com/Clever/go-bench
make
```

## Usage
You can run go-bench using:

```
go run bench.go
```

Or:

```
make build
./bin/bench
```

The following command-line flags are supported:

flag | required? | description
:---: | :---: | :---:
--speed | no; default 1 | Sets multiplier for playback speed
--output | no; not written if not provided | Path to file to which JSON output should be written
--root | yes | URL root for requests

go-bench reads requests to playback from standard input in the following format:

```
time,method,path,auth,extra
```

item | required? | description
:---: | :---: | :---:
time | yes | Time in ms after initialization to send request
method | yes | HTTP method to use for request
path | yes | Path for request
auth | no | Authentication header value (will be passed to server directly as given)
extra | no | Information about the request that will be written to the output file

If you need a simple server to test your usage of go-bench, you can start one using:

```
go run start_server.go
```

## Vendoring

Please view the [dev-handbook for instructions](https://github.com/Clever/dev-handbook/blob/master/golang/godep.md).
