package main

import "fmt"

func add(a int, b int) (int, int) {
	return a, a + b
}

func addplus(a, b, c int) (int, int) {
	return a, a + b + c
}
func addplusplus(a, b, c int) (res1 int, res2 int) {
	res1 = a
	res2 = a + b + c
	return
}
func main() {
	res1, res2 := add(100, 12)
	fmt.Println(res1, res2)

	res2, _ = addplus(1, 1, 1)
	fmt.Println(res2)

	res1, res2 = addplusplus(2, 2, 2)
	fmt.Println(res1, res2)
}
