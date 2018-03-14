// +build darwin freebsd linux netbsd openbsd

package termcursor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/SUSE/termui/termcursor"
)

var _ = Describe("cursor", func() {
	Describe("Up", func() {
		It("moves the cursor up N lines", func() {
			Expect(termcursor.Up(5)).To(Equal("\033[5A"))
		})
	})

	Describe("ClearToEndOfLine", func() {
		It("clears the line after the cursor", func() {
			Expect(termcursor.ClearToEndOfLine()).To(Equal("\033[0K"))
		})
	})

	Describe("ClearToEndOfDisplay", func() {
		It("clears everything below the cursor", func() {
			Expect(termcursor.ClearToEndOfDisplay()).To(Equal("\033[0J"))
		})
	})

	Describe("Show", func() {
		It("shows the cursor", func() {
			Expect(termcursor.Show()).To(Equal("\033[?25h"))
		})
	})

	Describe("Hide", func() {
		It("hides the cursor", func() {
			Expect(termcursor.Hide()).To(Equal("\033[?25l"))
		})
	})
})
