package util

import (
	"fmt"
	"os"
	"runtime/debug"
)

func FreakOut(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		debug.PrintStack()
		os.Exit(1)
	}
}
