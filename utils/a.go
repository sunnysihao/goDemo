package utils

import "fmt"

const Cfg = "haha"

type AStruct struct {
	Val int
}

func (as *AStruct) Add() int {
	r := 0
	for i := 0; i < 6; i++ {
		r += as.Val
	}
	return r
}
func init() {
	fmt.Println("init package utils")
}
