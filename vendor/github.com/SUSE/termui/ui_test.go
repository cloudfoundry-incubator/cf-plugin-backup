package termui

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/SUSE/termui/termpassword"
)

type testState struct {
	ui  *UI
	in  *bytes.Buffer
	out *bytes.Buffer
}

func setup() (*UI, *testState) {
	in, out := &bytes.Buffer{}, &bytes.Buffer{}
	ui := New(in, out, nil)
	return ui, &testState{in: in, out: out}
}

func TestNew(t *testing.T) {
	t.Parallel()

	ui := New(nil, nil, nil)

	// Check types
	var _ io.ReadWriter = ui
	var _ termpassword.Reader = ui.PasswordReader
}

func TestUI_Print(t *testing.T) {
	t.Parallel()

	ui, state := setup()
	one, three := "one", "three"
	two := 5

	ui.Print(one, two, three)
	if got := state.out.String(); got != fmt.Sprint(one, two, three) {
		t.Errorf("output was wrong: %q", got)
	}
}

func TestUI_Printf(t *testing.T) {
	t.Parallel()

	ui, state := setup()
	format, value := "test %s printf", "the"

	ui.Printf(format, value)
	if got := state.out.String(); got != fmt.Sprintf(format, value) {
		t.Errorf("output was wrong: %q", got)
	}
}

func TestUI_Println(t *testing.T) {
	t.Parallel()

	ui, state := setup()
	one, three := "one", "three"
	two := 5

	ui.Println(one, two, three)
	if got := state.out.String(); got != fmt.Sprintln(one, two, three) {
		t.Errorf("output was wrong: %q", got)
	}
}

func TestUI_Prompt(t *testing.T) {
	t.Parallel()

	ui, state := setup()

	// Fill the "stdin" buffer.
	text := "prompt answer"
	fmt.Fprintln(state.in, text)

	input := ui.Prompt("Enter your name")

	if got := input; got != text {
		t.Errorf("input was wrong: %q", got)
	}
	if got := state.out.String(); got != "Enter your name: " {
		t.Errorf("output was wrong: %q", got)
	}
}

func TestUI_PromptDefault(t *testing.T) {
	t.Parallel()

	ui, state := setup()

	// Fill the "stdin" buffer.
	text := "myname"
	fmt.Fprintln(state.in, text)

	input := ui.PromptDefault("Enter your name", "gopher")

	if got := input; got != text {
		t.Errorf("input was wrong: %q", got)
	}
	if got := state.out.String(); got != "Enter your name [gopher]: " {
		t.Errorf("output was wrong: %q", got)
	}
}

func TestUI_PromptDefault_DefaultValue(t *testing.T) {
	t.Parallel()

	ui, state := setup()

	input := ui.PromptDefault("Enter your name", "gopher")

	if got := input; got != "gopher" {
		t.Errorf("input was wrong: %q", got)
	}
	if got := state.out.String(); got != "Enter your name [gopher]: " {
		t.Errorf("output was wrong: %q", got)
	}
}
