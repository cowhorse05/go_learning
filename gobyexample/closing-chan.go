package main

import "fmt"

func main() {
	jobs := make(chan int, 5)
	done := make(chan bool)

	go func() {
		for {
			j, more := <-jobs
			if more {
				fmt.Println("received job", j)
			} else {
				fmt.Println("received all jobs")
				done <- true
				return
			}
		}
	}()

	// for j := 1; j <= 3; j++ {
	// 	//提前关闭会导致panic
	// 	if j == 2 {
	// 		close(jobs)
	// 	}
	// 	jobs <- j
	// 	fmt.Println("sent job", j)
	// }
	for j := 1; j <= 3; j++ {
		jobs <- j
		fmt.Println("sent job", j)
	}
	close(jobs)
	fmt.Println("sent all jobs")

	<-done
}
