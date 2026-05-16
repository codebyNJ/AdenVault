package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// ErrPromptAborted is returned when the user cancels a prompt
// (ctrl-c, esc).
var ErrPromptAborted = errors.New("aborted")

// passwordModel is a Bubble Tea model for a single password prompt.
// When Confirm is true it asks for the password twice and only
// resolves once the two inputs match.
type passwordModel struct {
	prompt  string
	confirm bool
	pw      textinput.Model
	confirmInput textinput.Model
	stage   int // 0 = first entry, 1 = confirmation
	err     string
	done    bool
	cancel  bool
}

func newPasswordModel(prompt string, confirm bool) passwordModel {
	pw := textinput.New()
	pw.Placeholder = "master password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.Focus()
	pw.CharLimit = 256
	pw.Width = 40
	pw.Prompt = "› "
	pw.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	pw.TextStyle = lipgloss.NewStyle().Foreground(ColorFg)

	c := textinput.New()
	c.Placeholder = "confirm password"
	c.EchoMode = textinput.EchoPassword
	c.EchoCharacter = '•'
	c.CharLimit = 256
	c.Width = 40
	c.Prompt = "› "
	c.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	c.TextStyle = lipgloss.NewStyle().Foreground(ColorFg)

	return passwordModel{
		prompt:       prompt,
		confirm:      confirm,
		pw:           pw,
		confirmInput: c,
	}
}

func (m passwordModel) Init() tea.Cmd { return textinput.Blink }

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancel = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.stage == 0 {
				if strings.TrimSpace(m.pw.Value()) == "" {
					m.err = "password cannot be empty"
					return m, nil
				}
				if !m.confirm {
					m.done = true
					return m, tea.Quit
				}
				m.stage = 1
				m.pw.Blur()
				m.confirmInput.Focus()
				m.err = ""
				return m, textinput.Blink
			}
			// stage 1 — confirmation
			if m.confirmInput.Value() != m.pw.Value() {
				m.err = "passwords do not match — try again"
				m.confirmInput.SetValue("")
				return m, nil
			}
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	if m.stage == 0 {
		m.pw, cmd = m.pw.Update(msg)
	} else {
		m.confirmInput, cmd = m.confirmInput.Update(msg)
	}
	return m, cmd
}

func (m passwordModel) View() string {
	var b strings.Builder
	b.WriteString(Banner("") + "\n\n")
	b.WriteString(Title.Render(m.prompt) + "\n")

	if m.stage == 0 {
		b.WriteString(m.pw.View())
	} else {
		b.WriteString(Muted.Render("password:") + "  " + lipgloss.NewStyle().Foreground(ColorMuted).Render(strings.Repeat("•", len(m.pw.Value()))) + "\n")
		b.WriteString(Prompt.Render("confirm:  ") + m.confirmInput.View())
	}

	if m.err != "" {
		b.WriteString("\n\n" + Errored.Render("✗ "+m.err))
	}
	b.WriteString("\n\n" + HelpBar.Render("enter: submit  •  esc: cancel"))
	return b.String()
}

// PromptPassword runs an interactive password prompt and returns the
// typed password as a byte slice.
//
// If stdin or stderr is not a TTY (e.g. piped from CI) and
// allowStdin is true, the password is read from stdin instead, as a
// single line. If allowStdin is false in that case, the function
// returns an error explaining how to use --password-stdin.
func PromptPassword(prompt string, confirm, allowStdin bool) ([]byte, error) {
	// Use stderr as the UI surface so stdout stays clean for piping.
	if !isInteractive() {
		if allowStdin {
			return readPasswordFromStdin()
		}
		return nil, errors.New("no terminal available — use --password-stdin to read from stdin")
	}

	m := newPasswordModel(prompt, confirm)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(os.Stdin))
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}
	fm := final.(passwordModel)
	if fm.cancel {
		return nil, ErrPromptAborted
	}
	return []byte(fm.pw.Value()), nil
}

// PromptPasswordStdin always reads the password from stdin. Used when
// --password-stdin is passed.
func PromptPasswordStdin() ([]byte, error) {
	return readPasswordFromStdin()
}

func readPasswordFromStdin() ([]byte, error) {
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read password from stdin: %w", err)
	}
	pw := strings.TrimRight(line, "\r\n")
	if pw == "" {
		return nil, errors.New("empty password on stdin")
	}
	return []byte(pw), nil
}

// isInteractive reports whether both stdin and stderr are attached to
// a TTY. Bubble Tea needs stdin for input and we render to stderr.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

// IsInteractive is the exported view of isInteractive — used by the
// cmd layer to decide whether to skip y/n confirmations when running
// non-interactively (e.g. with --password-stdin in CI).
func IsInteractive() bool { return isInteractive() }
