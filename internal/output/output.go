package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Printer handles all terminal output.
type Printer struct {
	w    io.Writer
	json bool
	ci   bool
}

// New creates a Printer. json=true emits machine-readable JSON.
// ci=true suppresses ANSI formatting.
func New(jsonMode, ciMode bool) *Printer {
	return &Printer{w: os.Stdout, json: jsonMode, ci: ciMode}
}

// ── ANSI helpers ──────────────────────────────────────────────────────────────

func (p *Printer) ansi(code string, s string) string {
	if p.ci || p.json {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func (p *Printer) green(s string) string  { return p.ansi("32", s) }
func (p *Printer) red(s string) string    { return p.ansi("31", s) }
func (p *Printer) yellow(s string) string { return p.ansi("33", s) }
func (p *Printer) bold(s string) string   { return p.ansi("1", s) }
func (p *Printer) dim(s string) string    { return p.ansi("2", s) }
func (p *Printer) cyan(s string) string   { return p.ansi("36", s) }

// ── Output methods ────────────────────────────────────────────────────────────

// Success prints a green success message (or JSON if in JSON mode).
func (p *Printer) Success(msg string) {
	if p.json {
		return
	}
	fmt.Fprintf(p.w, "%s %s\n", p.green("✓"), msg)
}

// Info prints an informational message.
func (p *Printer) Info(msg string) {
	if p.json {
		return
	}
	fmt.Fprintf(p.w, "%s %s\n", p.dim("›"), msg)
}

// Warn prints a yellow warning.
func (p *Printer) Warn(msg string) {
	if p.json {
		return
	}
	fmt.Fprintf(p.w, "%s %s\n", p.yellow("!"), msg)
}

// Error prints a red error to stderr.
func (p *Printer) Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", p.red("✗"), msg)
}

// Fatal prints an error and exits.
func (p *Printer) Fatal(msg string) {
	p.Error(msg)
	os.Exit(1)
}

// Header prints a bold section header.
func (p *Printer) Header(msg string) {
	if p.json {
		return
	}
	fmt.Fprintf(p.w, "\n%s\n", p.bold(msg))
}

// KeyValue prints a key: value pair, nicely aligned.
func (p *Printer) KeyValue(key, value string) {
	if p.json {
		return
	}
	fmt.Fprintf(p.w, "  %-18s %s\n", p.cyan(key+":"), value)
}

// JSON marshals v and prints it.
func (p *Printer) JSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		p.Error("Failed to marshal JSON: " + err.Error())
		return
	}
	fmt.Fprintf(p.w, "%s\n", b)
}

// Table prints a simple text table.
func (p *Printer) Table(headers []string, rows [][]string) {
	if p.json {
		return
	}
	// Compute column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	var sb strings.Builder
	for i, h := range headers {
		sb.WriteString(p.bold(fmt.Sprintf("%-*s", widths[i]+2, h)))
	}
	fmt.Fprintln(p.w, sb.String())

	// Separator
	sep := ""
	for _, w := range widths {
		sep += strings.Repeat("─", w+2)
	}
	fmt.Fprintln(p.w, p.dim(sep))

	// Rows
	for _, row := range rows {
		var line strings.Builder
		for i, cell := range row {
			if i < len(widths) {
				line.WriteString(fmt.Sprintf("%-*s", widths[i]+2, cell))
			}
		}
		fmt.Fprintln(p.w, line.String())
	}
}

// Progress prints a simple progress line (overwrites current line in terminal).
func (p *Printer) Progress(pct int, msg string) {
	if p.json {
		return
	}
	if p.ci {
		fmt.Fprintf(p.w, "%s %d%%\n", msg, pct)
		return
	}
	bar := strings.Repeat("█", pct/5) + strings.Repeat("░", 20-pct/5)
	fmt.Fprintf(p.w, "\r  [%s] %3d%%  %s", p.cyan(bar), pct, msg)
	if pct >= 100 {
		fmt.Fprintln(p.w)
	}
}

// Newline prints a blank line.
func (p *Printer) Newline() {
	if !p.json {
		fmt.Fprintln(p.w)
	}
}
