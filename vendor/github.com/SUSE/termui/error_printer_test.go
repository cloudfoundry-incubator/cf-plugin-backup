package termui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/fatih/color"
)

const (
	codeNotImplementedError = iota + 200
	codeUnsupportedOSError
)

type ErrorImpl struct {
	err  error
	code int
}

func (e ErrorImpl) Error() string {
	return e.err.Error()
}

func (e ErrorImpl) Code() int {
	return e.code
}

func UnsupportedOSError(os string) ErrorImpl {
	return ErrorImpl{
		err:  fmt.Errorf("Unsupported OS: %s", os),
		code: codeUnsupportedOSError,
	}
}

func NotImplementedError() ErrorImpl {
	return ErrorImpl{
		err:  fmt.Errorf("This feature has not been implemented"),
		code: codeNotImplementedError,
	}
}

func TestErrorPrinter_Printing(t *testing.T) {
	saveExitFunction := exitFunction
	defer func() {
		exitFunction = saveExitFunction
	}()

	exitCode := -1
	exitFunction = func(code int) {
		exitCode = codeUnsupportedOSError
	}

	in, out := &bytes.Buffer{}, &bytes.Buffer{}
	ui := New(in, out, nil)

	printer := NewErrorPrinter(ui)

	err1 := UnsupportedOSError("windows")
	err2 := fmt.Errorf("Generic error")
	printer.PrintAndExit(err1)

	exp := color.RedString("Error (%d): Unsupported OS: windows", codeUnsupportedOSError)
	if got := strings.TrimSpace(out.String()); got != exp {
		t.Error("wrong error message:", got)
	}
	if exitCode != codeUnsupportedOSError {
		t.Error("wrong exit code:", exitCode)
	}

	out.Reset()
	printer.PrintWarning(err2)
	exp = color.YellowString("Warning (%d): Generic error", CodeUnknownError)
	if got := strings.TrimSpace(out.String()); got != exp {
		t.Error("wrong error message:", got)
	}
}
