package main

import (
	"fmt"
	"time"
)

func main() {

	requests := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		requests <- i
	}
	close(requests)

	//平滑限流
	limiter := time.Tick(time.Second)

	for req := range requests {
		<-limiter //每次处理请求都要等待limiter
		fmt.Println("request", req, time.Now())
	}

	//1.带突发的速率限制，令牌桶算法，初始化三个令牌
	burstyLimiter := make(chan time.Time, 3)

	for i := 0; i < 3; i++ {
		burstyLimiter <- time.Now() //预先放入三个令牌
	}

	//2.后台协程每秒补充1个令牌，所以是一秒一个请求
	go func() {
		for t := range time.Tick(time.Second) {
			burstyLimiter <- t
		}
	}()

	burstyRequests := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		burstyRequests <- i
	}
	close(burstyRequests)
	for req := range burstyRequests {
		<-burstyLimiter
		fmt.Println("request", req, time.Now())
	}
}
