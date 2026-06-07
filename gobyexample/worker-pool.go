package main

import (
	"fmt"
	"time"
)

func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		fmt.Println("worker ", id, " started job ", j)
		time.Sleep(time.Second)
		fmt.Println("worker ", id, " finished job ", j)
		results <- 2 * j
	}
}

const numJobs = 5

func main() {

	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	//开了三个worker协程
	for w := 0; w < 3; w++ {
		go worker(w, jobs, results)
	}

	//然后生产者添加job
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	for a := 1; a <= numJobs; a++ {
		<-results
	}
}
