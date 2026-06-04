package main

import "fmt"

func main() {
	nums := []int{2, 3, 4}
	sum := 0
	for _, num := range nums {
		sum += num
	}
	fmt.Println(sum)

	kvs := map[string]string{"lyf": "wzq", "wzq": "lyf"}
	for k, v := range kvs {
		fmt.Printf("%s -> %s ", k, v)
	}
	fmt.Println()
	for k := range kvs {
		fmt.Println("key:", k)
	}

	for i, c := range "go" {
		fmt.Println(i, c)
	}
}
