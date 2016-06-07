package commands

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/cloudfoundry/cli/plugin"
)

//GetBearerToken - returns token from cf cli
func GetBearerToken(cliConnection plugin.CliConnection) (string, error) {
	token, err := cliConnection.AccessToken()
	if err != nil {
		return "", err
	}
	return strings.Replace(token, "bearer ", "", -1), nil
}

func getFileSha(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sha1 := sha1.New()
	_, err = io.Copy(sha1, f)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sha1.Sum(nil)), nil
}

func ShowFailed(message string) {
	traceEnv := os.Getenv("CF_TRACE")
	traceLogger := trace.NewLogger(Writer, false, traceEnv, "")

	ui := terminal.NewUI(os.Stdin, Writer, terminal.NewTeePrinter(Writer), traceLogger)
	ui.Failed(fmt.Sprintf("%s\n", message))
}

func ShowOK(message string) {
	traceEnv := os.Getenv("CF_TRACE")
	traceLogger := trace.NewLogger(Writer, false, traceEnv, "")

	ui := terminal.NewUI(os.Stdin, Writer, terminal.NewTeePrinter(Writer), traceLogger)
	ui.Say(terminal.SuccessColor("OK"))
	fmt.Printf("%s\n", message)
}

func Confirm(message string) bool {
	traceEnv := os.Getenv("CF_TRACE")
	traceLogger := trace.NewLogger(Writer, false, traceEnv, "")

	ui := terminal.NewUI(os.Stdin, Writer, terminal.NewTeePrinter(Writer), traceLogger)
	return ui.Confirm(message)
}
