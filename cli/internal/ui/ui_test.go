package ui

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func withUI(t *testing.T, enabled bool, fn func()) string {
	t.Helper()
	prevOut := Out
	prevEnabled := Enabled
	Enabled = enabled
	var buf bytes.Buffer
	Out = &buf
	t.Cleanup(func() {
		Out = prevOut
		Enabled = prevEnabled
	})
	fn()
	return buf.String()
}

func boxInnerWidth(line string) int {
	const prefix = "  "
	i := strings.Index(line, "│")
	if i < 0 {
		return -1
	}
	j := strings.LastIndex(line, "│")
	if j <= i {
		return -1
	}
	mid := stripANSI(line[i+len("│") : j])
	return utf8.RuneCountInString(mid)
}

func TestBoxAlignedBorders(t *testing.T) {
	out := withUI(t, true, func() {
		Box("Ready to ship", []string{
			Green("✔ ") + "everything checks out",
			Dim("run: ") + "pc ship --wait",
		})
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var top, bottom string
	var body []string
	for _, ln := range lines {
		if strings.Contains(ln, "╭") {
			top = ln
			continue
		}
		if strings.Contains(ln, "╰") {
			bottom = ln
			continue
		}
		if strings.Contains(ln, "│") {
			body = append(body, ln)
		}
	}
	if top == "" || bottom == "" {
		t.Fatalf("missing box borders:\n%s", out)
	}
	topInner := strings.Count(top, "─")
	bottomInner := strings.Count(bottom, "─")
	if topInner != bottomInner {
		t.Fatalf("top/bottom dash mismatch: top=%d bottom=%d\n%s", topInner, bottomInner, out)
	}

	wantInner := topInner
	for _, ln := range body {
		if got := boxInnerWidth(ln); got != wantInner {
			t.Fatalf("content width %d != border width %d in %q\n%s", got, wantInner, ln, out)
		}
	}
}

func TestHeadingIndent(t *testing.T) {
	out := withUI(t, true, func() {
		Heading("Checks")
	})
	for _, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if !strings.HasPrefix(ln, "  ") {
			t.Fatalf("heading should be indented, got %q", ln)
		}
		break
	}
}

func TestSpinnerPlainModeSingleLine(t *testing.T) {
	out := withUI(t, false, func() {
		sp := NewSpinner("Loading manifest…").Start()
		sp.Success("Context loaded")
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one line in plain mode, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "Context loaded") {
		t.Fatalf("expected success line, got %q", lines[0])
	}
}
