package main

import "fmt"

func main() {
	messages := make(chan string, 2)

	messages <- "ping"
	messages <- "base"

	fmt.Println((<-messages))
	fmt.Println((<-messages))

}
