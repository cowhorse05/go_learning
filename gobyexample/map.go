package main

import "fmt"

func main() {
	m := make(map[string]int)
	m["manba out"] = 1
	m["manba back"] = 0
	fmt.Println(m)

	m["manba out"] = 46
	fmt.Println(m)
	_, pr := m["manba out"]
	fmt.Println("pr:", pr)

	n := map[string]int{"foo": 1, "bar": 2}
	fmt.Println(n)
}
