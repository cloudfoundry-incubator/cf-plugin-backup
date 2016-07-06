package util

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"
)

func ConcatStringArray(lines []string) string {
	var buffer bytes.Buffer

	for _, line := range lines {
		buffer.WriteString(line)
	}

	return buffer.String()
}

func FreakOut(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		debug.PrintStack()
		os.Exit(1)
	}
}
