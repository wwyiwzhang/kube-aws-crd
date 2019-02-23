package main

import "fmt"

func main() {

	v := "hello world"
	printValue(v)

}

func printValue(v string) {
	fmt.Printf("Value is %s", v)
	c := nil
	fmt.Println(&v != nil)
	fmt.Println(c == nil)
}
