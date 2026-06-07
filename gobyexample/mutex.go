package main

import (
	"fmt"
	"sync"
)

type Container struct {
	mu       sync.Mutex
	counters map[string]int
}

func (c *Container) increase(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counters[name]++
}

func main() {
	c := Container{
		counters: map[string]int{"a": 0, "b": 0},
	}

	var wg sync.WaitGroup

	doIncre := func(name string, n int) {
		for i := 0; i < n; i++ {
			c.increase(name)
		}
		wg.Done()
	}

	wg.Add(3) //添加三个协程

	go doIncre("a", 1000)
	go doIncre("a", 1000)
	go doIncre("b", 1000)

	wg.Wait()
	fmt.Println(c.counters)
}
