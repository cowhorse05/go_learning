package main

import "fmt"

func add(a int, b int) int {
	return a + b
}

func addplus(a, b, c int) int {
	return a + b + c
}
func main() {
	res := add(100, 12)
	fmt.Println(res)

	res = addplus(1, 1, 1)
	fmt.Println(res)
}
