package sigint

import (
	"os"
	"testing"
)

func TestAdd(t *testing.T) {
	handler := NewHandler()
	if l := len(handler.callbacks); l != 0 {
		t.Error("no callbacks should be registered:", l)
	}

	handler.Add(func() {})

	if l := len(handler.callbacks); l != 1 {
		t.Error("exactly one callback should be registered:", l)
	}
}

func TestHandler_Exit(t *testing.T) {
	saveExitFunction := exitFunction
	defer func() {
		exitFunction = saveExitFunction
	}()

	exitFunctionCalled := false
	exitFunction = func(int) {
		exitFunctionCalled = true
	}

	sigintChan := make(chan os.Signal)
	handler := Handler{
		sigintChan: sigintChan,
	}
	handlerCalled := false
	handler.callbacks = append(handler.callbacks, func() {
		handlerCalled = true
	})

	handler.Exit(SigInt)
	if !exitFunctionCalled {
		t.Error("exit function should have been called")
	}
	if !handlerCalled {
		t.Error("exit callback should have been called")
	}
}
