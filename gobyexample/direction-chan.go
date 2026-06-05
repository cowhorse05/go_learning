package main

import "fmt"

func ping(ping chan string, msg string) {
	ping <- msg
}
func pong(pong chan string, ping chan string) {
	msg := <-ping
	pong <- msg
}
func main() {
	pings := make(chan string, 1)
	pongs := make(chan string, 1)

	ping(pings, "passed message")
	pong(pongs, pings)
	fmt.Println(<-pongs)
}
