package main

import "fmt"

type A struct {
	Name string
}

func (a *A) FuncA() {
	fmt.Println(a.Name)
}

var a *A

func main() {
	a.FuncA()
}
