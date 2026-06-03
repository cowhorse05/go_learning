package main
import "fmt"

func for_loop(){
	
	for i := 1; i < 10; i++{
		fmt.Println(i)
	}
	j := 1
	for{

		fmt.Println("loop")
		j ++
		if j > 10{
			break
		}
	}

	for n := 0; n <= 5; n++ {
        if n%2 == 0 {
            continue
        }
        fmt.Println(n)
    }
	
}

func main(){
	for_loop()
}