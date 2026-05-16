package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// SecretRow is the minimal data needed to render one line of the list
// view. We intentionally don't depend on internal/vault here so this
// file stays import-free apart from lipgloss.
type SecretRow struct {
	Name      string
	UpdatedAt time.Time
}

// RenderList returns a styled, boxed view of the project's secrets.
// project and env appear in the title bar; rows are zero or more
// entries. The output is suitable for printing to stderr (humans) or
// stdout (when the user explicitly wants pretty output).
func RenderList(project, env string, rows []SecretRow) string {
	var b strings.Builder
	b.WriteString(BigLogo())
	b.WriteByte('\n')
	pink := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	cyan := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	b.WriteString("  " +
		Muted.Render("project:") + " " + pink.Render(project) + "  " +
		Muted.Render("env:") + " " + cyan.Render(env) + "  " +
		Muted.Render("secrets:") + " " + Key.Render(fmt.Sprintf("%d", len(rows))) + "\n\n")
	b.WriteString(GradientRule(40) + "\n\n")

	if len(rows) == 0 {
		b.WriteString(Box.Render(
			Muted.Render("no secrets yet — store one with ") +
				Key.Render("adenV set KEY value")))
		return b.String()
	}

	nameW := len("KEY")
	for _, r := range rows {
		if l := len(r.Name); l > nameW {
			nameW = l
		}
	}

	header := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Bold(true).
		Render(pad("KEY", nameW)+"   UPDATED")

	var body strings.Builder
	body.WriteString(header + "\n")
	body.WriteString(Muted.Render(strings.Repeat("─", nameW+3+len("2026-05-16 12:00"))) + "\n")
	for _, r := range rows {
		body.WriteString(Key.Render(pad(r.Name, nameW)))
		body.WriteString("   ")
		body.WriteString(Muted.Render(r.UpdatedAt.Local().Format("2006-01-02 15:04")))
		body.WriteString("\n")
	}
	body.WriteString("\n" + Muted.Render(fmt.Sprintf("%d secret(s) in %s vault", len(rows), env)))

	b.WriteString(Box.Render(body.String()))
	return b.String()
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// PrintSuccess writes a green check + message to w.
func PrintSuccess(w io.Writer, msg string) {
	fmt.Fprintln(w, Success.Render("✓ ")+msg)
}

// PrintInfo writes a bullet + message to w.
func PrintInfo(w io.Writer, msg string) {
	fmt.Fprintln(w, Bullet()+" "+msg)
}

// PrintWarn writes a yellow warning to w.
func PrintWarn(w io.Writer, msg string) {
	fmt.Fprintln(w, Warn.Render("! ")+msg)
}

// PrintError writes a red error to w.
func PrintError(w io.Writer, msg string) {
	fmt.Fprintln(w, Errored.Render("✗ ")+msg)
}
