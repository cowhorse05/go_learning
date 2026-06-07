package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int) {
	fmt.Printf("worker %d staring\n", id)
	time.Sleep(time.Second)
	fmt.Printf("worker %d done\n", id)
}

func main() {
	var wg sync.WaitGroup

	for i := 1; i < 6; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			worker(i)
		}()
	}
	wg.Wait()
}
