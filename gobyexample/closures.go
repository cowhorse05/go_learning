// cpp只能伪闭包，你小子有真闭包
package main

import "fmt"

func inclose() func() int {
	i := 0
	return func() int {
		i++
		return i
	}
}

func main() {
	nextIn := inclose()

	//如果多个go routine访问，则不是线程安全的
	fmt.Println(nextIn())
	fmt.Println(nextIn())
	fmt.Println(nextIn())

	newNextIn := inclose()
	fmt.Println(newNextIn())
}
