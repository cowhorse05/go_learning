package main

import "fmt"

// 阶乘函数（Factorial）
func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}

// 斐波那契函数（Fibonacci）- 函数式定义
var fibonacci func(n int) int

func main() {
	// 计算阶乘
	fmt.Println("7! =", factorial(7))

	// 定义斐波那契函数（递归引用自身）
	fibonacci = func(n int) int {
		if n < 2 {
			return n
		}
		return fibonacci(n-1) + fibonacci(n-2)
	}

	// 计算斐波那契数列
	fmt.Println("Fibonacci(7) =", fibonacci(7))

	// 打印斐波那契数列前10项
	fmt.Print("斐波那契数列前10项: ")
	for i := 0; i < 10; i++ {
		fmt.Printf("%d ", fibonacci(i))
	}
	fmt.Println()
}
