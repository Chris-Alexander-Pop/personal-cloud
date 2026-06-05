package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Details is an aligned key/value panel.
type Details struct {
	rows [][2]string
}

// NewDetails returns an empty detail panel.
func NewDetails() *Details { return &Details{} }

// Add appends a key/value row. Empty values are skipped.
func (d *Details) Add(key, value string) *Details {
	if strings.TrimSpace(value) == "" {
		return d
	}
	d.rows = append(d.rows, [2]string{key, value})
	return d
}

// Render prints the aligned panel with dim keys and a faint gutter.
func (d *Details) Render() {
	if len(d.rows) == 0 {
		return
	}
	width := 0
	for _, r := range d.rows {
		if w := utf8.RuneCountInString(r[0]); w > width {
			width = w
		}
	}
	gutter := Dim(sym("│", "|"))
	for _, r := range d.rows {
		pad := strings.Repeat(" ", width-utf8.RuneCountInString(r[0]))
		fmt.Fprintf(Out, "  %s %s%s  %s\n", gutter, Dim(r[0]), pad, r[1])
	}
}

// Box renders a titled box around the given content lines.
func Box(title string, lines []string) {
	fmt.Fprintln(Out)
	if !Enabled {
		if title != "" {
			fmt.Fprintf(Out, "[ %s ]\n", title)
		}
		for _, ln := range lines {
			fmt.Fprintf(Out, "  %s\n", stripANSI(ln))
		}
		return
	}

	width := utf8.RuneCountInString(stripANSI(title))
	for _, ln := range lines {
		if w := utf8.RuneCountInString(stripANSI(ln)); w > width {
			width = w
		}
	}
	inner := width + 2
	tl, tr, bl, br := "╭", "╮", "╰", "╯"
	h, v := "─", "│"

	top := cBrand + tl + strings.Repeat(h, inner) + tr + cReset
	if title != "" {
		t := " " + Bold(title) + " "
		dashes := inner - utf8.RuneCountInString(stripANSI(t))
		if dashes < 0 {
			dashes = 0
		}
		top = cBrand + tl + h + cReset + t + cBrand + strings.Repeat(h, dashes-1) + tr + cReset
	}
	fmt.Fprintln(Out, "  "+top)
	for _, ln := range lines {
		pad := inner - 1 - utf8.RuneCountInString(stripANSI(ln))
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(Out, "  %s %s%s %s\n", cBrand+v+cReset, ln, strings.Repeat(" ", pad), cBrand+v+cReset)
	}
	fmt.Fprintln(Out, "  "+cBrand+bl+strings.Repeat(h, inner)+br+cReset)
}

// stripANSI removes escape sequences so visible width can be measured.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\033' {
			inEsc = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
