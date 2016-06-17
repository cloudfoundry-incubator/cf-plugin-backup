package util

import (
	"fmt"
	"os"
)

func FreakOut(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(1)
	}
}
