package main

import "fmt"

func main() {
	queue := make(chan string, 2)
	queue <- "one"
	queue <- "two"
	close(queue)

	for ele := range queue {
		fmt.Println(ele)
	}
}
