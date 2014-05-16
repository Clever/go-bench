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
	_, err2 := fmt.Scanf("%s", &input)
	if (err != nil) {
		fmt.Println(err2)
	}
	fmt.Println("Exiting.")
}
