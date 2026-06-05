// Package ui renders the pc CLI: colors, spinners, badges, detail panels
// and boxes. It is self-contained (no external deps) and degrades gracefully
// when stdout is not a TTY or when NO_COLOR is set.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Out and Err are the destinations for normal and error output.
var (
	Out io.Writer = os.Stdout
	Err io.Writer = os.Stderr
)

// Enabled reports whether ANSI styling and animations are used. It is set in
// init based on TTY detection and the NO_COLOR / TERM environment variables.
var Enabled = detectColor()

func detectColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if v := os.Getenv("PC_FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(os.Stdout)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// 256-color palette tuned for a clean, modern terminal look.
const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[2m"
	cItalic = "\033[3m"
	cUnder  = "\033[4m"
	cBrand  = "\033[38;5;39m"  // deep sky blue
	cBrand2 = "\033[38;5;75m"  // lighter blue
	cGreen  = "\033[38;5;78m"  // mint green
	cRed    = "\033[38;5;203m" // coral red
	cYellow = "\033[38;5;221m" // soft yellow
	cOrange = "\033[38;5;215m" // amber
	cGray   = "\033[38;5;244m" // mid gray
	cGrayLo = "\033[38;5;240m" // dim gray
	cPurple = "\033[38;5;141m" // lavender
	cCyan   = "\033[38;5;80m"  // teal cyan
	cWhite  = "\033[38;5;255m" // near white
)

func wrap(code, s string) string {
	if !Enabled {
		return s
	}
	return code + s + cReset
}

// Style helpers.
func Bold(s string) string   { return wrap(cBold, s) }
func Dim(s string) string    { return wrap(cDim, s) }
func Italic(s string) string { return wrap(cItalic, s) }
func Brand(s string) string  { return wrap(cBrand, s) }
func Brand2(s string) string { return wrap(cBrand2, s) }
func Green(s string) string  { return wrap(cGreen, s) }
func Red(s string) string    { return wrap(cRed, s) }
func Yellow(s string) string { return wrap(cYellow, s) }
func Orange(s string) string { return wrap(cOrange, s) }
func Gray(s string) string   { return wrap(cGray, s) }
func Purple(s string) string { return wrap(cPurple, s) }
func Cyan(s string) string   { return wrap(cCyan, s) }
func White(s string) string  { return wrap(cWhite, s) }

func BoldBrand(s string) string {
	if !Enabled {
		return s
	}
	return cBold + cBrand + s + cReset
}

// Symbols (fall back to ASCII when styling is disabled).
func sym(fancy, plain string) string {
	if Enabled {
		return fancy
	}
	return plain
}

func tick() string  { return sym("✔", "OK") }
func cross() string { return sym("✗", "X") }
func dot() string   { return sym("•", "*") }
func arrow() string { return sym("→", "->") }

// Logo prints the pc banner with a tagline.
func Logo(tagline string) {
	if !Enabled {
		fmt.Fprintln(Out, "pc — "+tagline)
		return
	}
	bar := cBrand + "▌" + cReset
	fmt.Fprintln(Out)
	fmt.Fprintf(Out, "  %s %s%spc%s %s\n", bar, cBold, cBrand, cReset, Dim("personal-cloud"))
	fmt.Fprintf(Out, "  %s %s\n", bar, Dim(tagline))
	fmt.Fprintln(Out)
}

// Heading prints a section heading with a leading brand bar.
func Heading(title string) {
	fmt.Fprintln(Out)
	if !Enabled {
		fmt.Fprintf(Out, "== %s ==\n", title)
		return
	}
	fmt.Fprintf(Out, "%s▌%s %s%s%s\n", cBrand, cReset, cBold, title, cReset)
}

// Rule prints a faint horizontal divider.
func Rule() {
	fmt.Fprintln(Out, Dim(strings.Repeat("─", 52)))
}

// Step prints an in-progress style line (used when no spinner is wanted).
func Step(format string, a ...any) {
	fmt.Fprintf(Out, "  %s %s\n", Brand(arrow()), fmt.Sprintf(format, a...))
}

// Success prints a green check line.
func Success(format string, a ...any) {
	fmt.Fprintf(Out, "  %s %s\n", Green(tick()), fmt.Sprintf(format, a...))
}

// Fail prints a red cross line.
func Fail(format string, a ...any) {
	fmt.Fprintf(Out, "  %s %s\n", Red(cross()), fmt.Sprintf(format, a...))
}

// Warn prints a yellow warning line.
func Warn(format string, a ...any) {
	fmt.Fprintf(Out, "  %s %s\n", Yellow(sym("!", "!")), fmt.Sprintf(format, a...))
}

// Info prints a dim informational line.
func Info(format string, a ...any) {
	fmt.Fprintf(Out, "  %s %s\n", Gray(dot()), Dim(fmt.Sprintf(format, a...)))
}

// Bullet prints a plain bullet line.
func Bullet(format string, a ...any) {
	fmt.Fprintf(Out, "  %s %s\n", Brand(dot()), fmt.Sprintf(format, a...))
}

// ErrorBlock renders a prominent error to Err.
func ErrorBlock(msg string) {
	fmt.Fprintln(Err)
	if !Enabled {
		fmt.Fprintf(Err, "ERROR: %s\n", msg)
		return
	}
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	fmt.Fprintf(Err, "  %s%s %s%s\n", cRed, cross(), cBold, "Failed"+cReset)
	for _, ln := range lines {
		fmt.Fprintf(Err, "  %s %s\n", cRed+"│"+cReset, ln)
	}
	fmt.Fprintln(Err)
}

// Link renders an underlined, cyan URL.
func Link(url string) string {
	if !Enabled {
		return url
	}
	return cUnder + cCyan + url + cReset
}

// LinkLine prints a labelled link row.
func LinkLine(label, url string) {
	fmt.Fprintf(Out, "  %s %s\n", Dim(label), Link(url))
}

// Badge renders a filled status chip for a pipeline/step state.
func Badge(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	label := strings.ToUpper(state)
	if label == "" {
		label = "PENDING"
		state = "pending"
	}
	if !Enabled {
		return "[" + label + "]"
	}
	var bg string
	switch state {
	case "success":
		bg = "\033[48;5;78m"
	case "running", "started":
		bg = "\033[48;5;39m"
	case "pending", "waiting", "waiting_on_deps", "blocked":
		bg = "\033[48;5;221m"
	case "skipped":
		bg = "\033[48;5;244m"
	case "failure", "error", "killed", "declined":
		bg = "\033[48;5;203m"
	default:
		bg = "\033[48;5;244m"
	}
	// dark text on colored background for contrast
	return bg + "\033[38;5;235m" + cBold + " " + label + " " + cReset
}

// StateDot renders a colored dot followed by the (lowercased) state label.
func StateDot(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		state = "pending"
	}
	var col func(string) string
	switch state {
	case "success":
		col = Green
	case "running", "started":
		col = Brand
	case "pending", "waiting", "waiting_on_deps", "blocked":
		col = Yellow
	case "skipped":
		col = Gray
	case "failure", "error", "killed", "declined":
		col = Red
	default:
		col = Gray
	}
	return col(dot()) + " " + col(state)
}
