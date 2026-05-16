// Package ui contains the Lip Gloss style definitions and the Bubble
// Tea models used by the aden CLI. Anything pretty lives here.
//
// The package is careful to send interactive UI to stderr so that
// commands like `aden get` and `aden export` keep stdout reserved for
// machine-readable output.
package ui

import "github.com/charmbracelet/lipgloss"

// Brand palette. Kept small on purpose — the goal is to feel
// intentional, not flashy.
var (
	ColorPrimary = lipgloss.Color("#7C3AED") // violet
	ColorAccent  = lipgloss.Color("#06B6D4") // cyan
	ColorSuccess = lipgloss.Color("#22C55E") // green
	ColorWarn    = lipgloss.Color("#F59E0B") // amber
	ColorError   = lipgloss.Color("#EF4444") // red
	ColorMuted   = lipgloss.Color("#6B7280") // grey
	ColorFg      = lipgloss.Color("#E5E7EB")
)

var (
	// Logo renders the aden wordmark at the top of interactive flows.
	Logo = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	Tagline = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true)

	Title = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginBottom(1)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	Success = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	Warn = lipgloss.NewStyle().
		Foreground(ColorWarn).
		Bold(true)

	Errored = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Key = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	Prompt = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	HelpBar = lipgloss.NewStyle().
		Foreground(ColorMuted)
)

// Banner returns the adenVault header used at the top of interactive
// screens (password prompt, confirm dialog, etc.). For the full
// gradient splash see banner.go::Splash.
func Banner(subtitle string) string { return MiniBanner(subtitle) }

// CheckMark returns a green check symbol.
func CheckMark() string { return Success.Render("✓") }

// Cross returns a red x symbol.
func Cross() string { return Errored.Render("✗") }

// Bullet returns a muted middle-dot.
func Bullet() string { return Muted.Render("•") }
