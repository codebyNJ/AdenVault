package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// EntryMeta is the plaintext metadata shown in the list view.
type EntryMeta struct {
	Label     string
	HasUser   bool
	HasURL    bool
	HasNotes  bool
	UpdatedAt time.Time
}

// RenderList returns a styled, boxed list of vault entries.
func RenderList(project, env string, rows []EntryMeta) string {
	var b strings.Builder
	b.WriteString(BigLogo())
	b.WriteByte('\n')

	pink := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	cyan := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

	countStyle := Key
	if len(rows) == 0 {
		countStyle = Warn
	}
	b.WriteString("  " +
		Muted.Render("vault:") + " " + pink.Render(project) + "  " +
		Muted.Render("env:") + " " + cyan.Render(env) + "  " +
		Muted.Render("entries:") + " " + countStyle.Render(fmt.Sprintf("%d", len(rows))) + "\n\n")
	b.WriteString(GradientRule(40) + "\n\n")

	if len(rows) == 0 {
		emptyTitle := Warn.Render("vault is empty")
		hint1 := Muted.Render("store your first password:")
		hint2 := "  " + Key.Render("adenV add github")
		hint3 := "  " + Key.Render("adenV add gmail")
		hint4 := Muted.Render("then run ") + Key.Render("adenV list") + Muted.Render(" again")
		b.WriteString(Box.Render(
			emptyTitle + "\n\n" +
				hint1 + "\n" +
				hint2 + "\n" +
				hint3 + "\n\n" +
				hint4,
		))
		return b.String()
	}

	labelW := len("LABEL")
	for _, r := range rows {
		if l := len(r.Label); l > labelW {
			labelW = l
		}
	}

	header := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true).
		Render(pad(strings.ToUpper("label"), labelW) + "   FIELDS   UPDATED")

	var body strings.Builder
	body.WriteString(header + "\n")
	body.WriteString(Muted.Render(strings.Repeat("─", labelW+3+8+3+len("2026-05-16 12:00"))) + "\n")

	for _, r := range rows {
		// Indicator icons: show which fields are populated.
		var icons strings.Builder
		if r.HasUser {
			icons.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Render("u"))
		} else {
			icons.WriteString(Muted.Render("·"))
		}
		if r.HasURL {
			icons.WriteString(lipgloss.NewStyle().Foreground(ColorSuccess).Render("w"))
		} else {
			icons.WriteString(Muted.Render("·"))
		}
		if r.HasNotes {
			icons.WriteString(lipgloss.NewStyle().Foreground(ColorWarn).Render("n"))
		} else {
			icons.WriteString(Muted.Render("·"))
		}

		body.WriteString(Key.Render(pad(r.Label, labelW)))
		body.WriteString("   " + icons.String() + "       ")
		body.WriteString(Muted.Render(r.UpdatedAt.Local().Format("2006-01-02 15:04")))
		body.WriteString("\n")
	}

	body.WriteString("\n" + Muted.Render(fmt.Sprintf("%d entr%s in %s vault", len(rows), plural(len(rows)), env)))
	body.WriteString("\n" + Muted.Render("icons: ") +
		lipgloss.NewStyle().Foreground(ColorAccent).Render("u") + Muted.Render("sername  ") +
		lipgloss.NewStyle().Foreground(ColorSuccess).Render("w") + Muted.Render("ebsite  ") +
		lipgloss.NewStyle().Foreground(ColorWarn).Render("n") + Muted.Render("otes"))

	b.WriteString(Box.Render(body.String()))
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func PrintSuccess(w io.Writer, msg string) { fmt.Fprintln(w, Success.Render("✓ ")+msg) }
func PrintInfo(w io.Writer, msg string)    { fmt.Fprintln(w, Bullet()+" "+msg) }
func PrintWarn(w io.Writer, msg string)    { fmt.Fprintln(w, Warn.Render("! ")+msg) }
func PrintError(w io.Writer, msg string)   { fmt.Fprintln(w, Errored.Render("✗ ")+msg) }
