package main

import (
	"flag"
	"fmt"
)

var funcName string

func init() {
	flag.StringVar(&funcName, "func", "", "the name of function to invoke")
}

func main() {
	flag.Parse()

	switch funcName {
	case "SendSMSTest":
		SendSMSTest()
	default:
		fmt.Println("unknow function name:", funcName)
	}
}
