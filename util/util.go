package util

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
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

// CheckUserScope takes a jwt token and checks for a given uaa scope
func CheckUserScope(jwtToken, scope string) (bool, error) {

	hasScope := false

	payload := strings.Split(jwtToken, ".")[1]
	b, err := base64.URLEncoding.DecodeString(payload)

	if err != nil {
		return hasScope, err
	}

	decoder := json.NewDecoder(bytes.NewBuffer(b))

	var m interface{}

	err = decoder.Decode(&m)
	if err != nil {
		return hasScope, err
	}
	t := m.(map[string]interface{})
	scopes := t["scope"].([]interface{})

	for _, s := range scopes {
		if s.(string) == scope {
			hasScope = true
			break
		}
	}

	return hasScope, nil
}
