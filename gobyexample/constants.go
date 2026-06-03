package main
import (
    "fmt"
    "math"
)

const s string = "const string ,can not be changed"

func main(){
	fmt.Println(s)

	const a = 5e10
	fmt.Println(a)

	const b = 30e10 / a
	fmt.Println(b)

	fmt.Println(math.Sin(b))
}