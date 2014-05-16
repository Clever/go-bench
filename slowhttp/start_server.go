package main

import (
	"fmt"
)

func main() {
	server, err := StartServer()
	if (err != nil) {
		panic(err)
	}
	defer server.Close()
	
	// continue running until user input given
	var input string
	if _, err := fmt.Scanf("%s", &input); err != nil {
		panic(err)
	}
	fmt.Println("Exiting.")
}
