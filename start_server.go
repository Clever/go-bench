package main

import (
	"github.com/Clever/go-bench/slowhttp"
	"fmt"
)

func main() {
	server, err := slowhttp.StartServer()
	if (err != nil) {
		panic(err)
	}
	defer server.Close()
	
	// continue running until user input given
	var input string
	_, err2 := fmt.Scanf("%s", &input)
	if (err != nil) {
		fmt.Println(err2)
	}
	fmt.Println("Exiting.")
	server.Close()
}
