package ohai

import (
	"fmt"
	"io"
)

// Ohai displays an informative message.
func Ohai(a ...interface{}) (int, error) {
	return Ohaif("%s", a...)
}

// Ohaif displays an informative message.
func Ohaif(format string, a ...interface{}) (int, error) {
	return fmt.Printf(fmt.Sprintf("==> %s", format), a...)
}

// Ohailn displays an informative message.
func Ohailn(a ...interface{}) (int, error) {
	return Ohaif("%s\n", a...)
}

// Fohai displays an informative message.
func Fohai(w io.Writer, a ...interface{}) (int, error) {
	return Fohaif(w, "%s", a...)
}

// Fohaif displays an informative message.
func Fohaif(w io.Writer, format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(w, fmt.Sprintf("==> %s", format), a...)
}

// Fohailn displays an informative message.
func Fohailn(w io.Writer, a ...interface{}) (int, error) {
	return Fohaif(w, "%s\n", a...)
}

// Success displays a success message.
func Success(a ...interface{}) (int, error) {
	return Successf("%s", a...)
}

// Successf displays a success message.
func Successf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(fmt.Sprintf("✓✓✓ %s", format), a...)
}

// Successln displays a success message.
func Successln(a ...interface{}) (int, error) {
	return Successf("%s\n", a...)
}

// Fsuccess displays an informative message.
func Fsuccess(w io.Writer, a ...interface{}) (int, error) {
	return Fsuccessf(w, "%s", a...)
}

// Fsuccessf displays an informative message.
func Fsuccessf(w io.Writer, format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(w, fmt.Sprintf("==> %s", format), a...)
}

// Fsuccessln displays an informative message.
func Fsuccessln(w io.Writer, a ...interface{}) (int, error) {
	return Fsuccessf(w, "%s\n", a...)
}

// Warning displays a warning message.
func Warning(a ...interface{}) (int, error) {
	return Warningf("%s", a...)
}

// Warningf displays a warning message.
func Warningf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(fmt.Sprintf("!!! %s", format), a...)
}

// Warningln displays a warning message.
func Warningln(a ...interface{}) (int, error) {
	return Warningf("%s\n", a...)
}

// Fwarning displays an informative message.
func Fwarning(w io.Writer, a ...interface{}) (int, error) {
	return Fwarningf(w, "%s", a...)
}

// Fwarningf displays an informative message.
func Fwarningf(w io.Writer, format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(w, fmt.Sprintf("==> %s", format), a...)
}

// Fwarningln displays an informative message.
func Fwarningln(w io.Writer, a ...interface{}) (int, error) {
	return Fwarningf(w, "%s\n", a...)
}
