package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// adenLogo is the ANSI-Shadow block lettering for the wordmark.
// Each character is colourised individually so we can paint a smooth
// horizontal gradient across all six rows.
var adenLogo = []string{
	` █████╗ ██████╗ ███████╗███╗   ██╗██╗   ██╗`,
	`██╔══██╗██╔══██╗██╔════╝████╗  ██║██║   ██║`,
	`███████║██║  ██║█████╗  ██╔██╗ ██║██║   ██║`,
	`██╔══██║██║  ██║██╔══╝  ██║╚██╗██║╚██╗ ██╔╝`,
	`██║  ██║██████╔╝███████╗██║ ╚████║ ╚████╔╝ `,
	`╚═╝  ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═══╝  ╚═══╝  `,
}

// Gradient stops for the wordmark and separator: pink → violet → cyan.
var gradientStops = []rgb{
	parseHex("#F472B6"),
	parseHex("#A78BFA"),
	parseHex("#22D3EE"),
}

type rgb struct{ r, g, b uint8 }

func parseHex(h string) rgb {
	h = strings.TrimPrefix(h, "#")
	r, _ := strconv.ParseUint(h[0:2], 16, 8)
	g, _ := strconv.ParseUint(h[2:4], 16, 8)
	b, _ := strconv.ParseUint(h[4:6], 16, 8)
	return rgb{uint8(r), uint8(g), uint8(b)}
}

func (c rgb) hex() string { return fmt.Sprintf("#%02X%02X%02X", c.r, c.g, c.b) }

func lerpRGB(a, b rgb, t float64) rgb {
	return rgb{
		r: uint8(float64(a.r) + (float64(b.r)-float64(a.r))*t),
		g: uint8(float64(a.g) + (float64(b.g)-float64(a.g))*t),
		b: uint8(float64(a.b) + (float64(b.b)-float64(a.b))*t),
	}
}

// gradientAt returns the colour at position t (0..1) along the
// multi-stop gradient.
func gradientAt(stops []rgb, t float64) rgb {
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	seg := t * float64(len(stops)-1)
	i := int(seg)
	return lerpRGB(stops[i], stops[i+1], seg-float64(i))
}

// BigLogo renders the multi-line ADEN wordmark with a horizontal
// pink → violet → cyan gradient.
func BigLogo() string {
	width := 0
	for _, l := range adenLogo {
		if w := lipgloss.Width(l); w > width {
			width = w
		}
	}
	if width < 1 {
		width = 1
	}

	var b strings.Builder
	for _, line := range adenLogo {
		col := 0
		for _, r := range line {
			if r == ' ' {
				b.WriteRune(' ')
				col++
				continue
			}
			t := float64(col) / float64(width-1)
			c := gradientAt(gradientStops, t)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c.hex())).Render(string(r)))
			col++
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// GradientRule prints a horizontal rule of n characters with the same
// pink → violet → cyan gradient as the wordmark.
func GradientRule(n int) string {
	if n < 1 {
		n = 1
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1)
		c := gradientAt(gradientStops, t)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c.hex())).Render("─"))
	}
	return b.String()
}

// SplashInfo carries the dynamic status shown beneath the wordmark.
// An empty Project means "no vault exists for this directory yet".
type SplashInfo struct {
	Project string
	Env     string
	Count   int
	Exists  bool
}

// Splash renders the full landing screen: gradient wordmark, tagline,
// status line, gradient separator, and a help footer. Designed to be
// printed to stderr (so stdout stays clean for piping).
func Splash(info SplashInfo) string {
	// rule width: align with the wordmark.
	ruleW := 0
	for _, l := range adenLogo {
		if w := lipgloss.Width(l); w > ruleW {
			ruleW = w
		}
	}
	if ruleW < 56 {
		ruleW = 56
	}

	var b strings.Builder
	b.WriteString(BigLogo())
	b.WriteByte('\n')

	// Subtitle: bolded keywords in pink + cyan, rest in muted.
	pink := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	cyan := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	b.WriteString("  " + pink.Render("adenVault") + "  " +
		Muted.Render("· no cloud. no subscription. no breach.") + "\n")
	b.WriteString("  " + Muted.Render("passwords stay on your machine  ·  ") +
		cyan.Render("zero network") +
		Muted.Render("  ·  one binary") + "\n\n")

	// Status line — equivalent to the screenshot's "session / turn / nodes".
	if info.Exists {
		count := strconv.Itoa(info.Count)
		b.WriteString("  " +
			Muted.Render("vault:") + " " + Key.Render(info.Project) + "   " +
			Muted.Render("env:") + " " + Key.Render(info.Env) + "   " +
			Muted.Render("entries:") + " " + Key.Render(count) + "\n")
	} else {
		b.WriteString("  " +
			Muted.Render("no vault here yet — run ") +
			Key.Render("adenV init") +
			Muted.Render(" to seal one") + "\n")
	}

	b.WriteByte('\n')
	b.WriteString(GradientRule(ruleW) + "\n\n")

	// Help hints (mirrors the screenshot's tip lines).
	b.WriteString("  " + Muted.Render("your passwords are one command away") + "\n")
	b.WriteString("  " + Key.Render("adenV --help") + Muted.Render("  for the full command list") + "\n")
	b.WriteString("  " + Key.Render("adenV i") + Muted.Render(" init  ·  ") +
		Key.Render("adenV add github") + Muted.Render(" store  ·  ") +
		Key.Render("adenV cp github") + Muted.Render(" copy") + "\n")
	return b.String()
}

// MiniBanner renders a one-line gradient header — used at the top of
// password prompts and confirms where the full splash would be overkill
// but we still want the brand cue.
func MiniBanner(subtitle string) string {
	pink := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	cyan := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	line := pink.Render("adenVault") + "  " +
		cyan.Render("·") + "  " +
		Muted.Render("sealed locally · unlocked by you")
	if subtitle != "" {
		line += "\n" + Muted.Render(subtitle)
	}
	return line
}
