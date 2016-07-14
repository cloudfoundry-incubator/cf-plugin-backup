package util

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"
)

// ConcatStringArray concatenates a string array
func ConcatStringArray(lines []string) string {
	var buffer bytes.Buffer

	for _, line := range lines {
		buffer.WriteString(line)
	}

	return buffer.String()
}

// FreakOut logs error and exits the program with exit code 1
func FreakOut(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		debug.PrintStack()
		os.Exit(1)
	}
}
