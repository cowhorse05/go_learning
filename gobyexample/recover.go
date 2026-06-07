package main

import "fmt"

func mayPanic() {
	panic("a problem")
}
func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recover. Error :\n", r)
		}
	}()

	mayPanic()
	fmt.Println("Afer myPanic()") //不会执行，因为main函数在panic点停止，并在继续处理完defer后终止执行
}
