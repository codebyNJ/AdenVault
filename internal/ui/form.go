package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EntryForm holds the result of a completed Add/Edit form.
type EntryForm struct {
	Label    string
	Username string
	Password string
	URL      string
	Notes    string
}

type formField int

const (
	fieldUsername formField = iota
	fieldPassword
	fieldURL
	fieldNotes
	fieldCount
)

type entryFormModel struct {
	label   string
	inputs  [fieldCount]textinput.Model
	focused formField
	done    bool
	cancel  bool
	err     string
}

func newEntryFormModel(label string, prefill EntryForm) entryFormModel {
	mkInput := func(placeholder string, echo textinput.EchoMode) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.EchoMode = echo
		if echo == textinput.EchoPassword {
			t.EchoCharacter = '•'
		}
		t.CharLimit = 512
		t.Width = 44
		t.Prompt = "  "
		t.PromptStyle = lipgloss.NewStyle().Foreground(ColorMuted)
		t.TextStyle = lipgloss.NewStyle().Foreground(ColorFg)
		t.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorMuted)
		return t
	}

	m := entryFormModel{label: label}
	m.inputs[fieldUsername] = mkInput("email or username", textinput.EchoNormal)
	m.inputs[fieldPassword] = mkInput("password", textinput.EchoPassword)
	m.inputs[fieldURL] = mkInput("https://... (optional)", textinput.EchoNormal)
	m.inputs[fieldNotes] = mkInput("notes (optional)", textinput.EchoNormal)

	// Pre-fill when editing an existing entry.
	m.inputs[fieldUsername].SetValue(prefill.Username)
	m.inputs[fieldPassword].SetValue(prefill.Password)
	m.inputs[fieldURL].SetValue(prefill.URL)
	m.inputs[fieldNotes].SetValue(prefill.Notes)

	m.inputs[fieldUsername].Focus()
	m.inputs[fieldUsername].PromptStyle = Prompt
	return m
}

func (m entryFormModel) Init() tea.Cmd { return textinput.Blink }

func (m entryFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit

		case "ctrl+s", "ctrl+d":
			if m.inputs[fieldPassword].Value() == "" {
				m.err = "password cannot be empty"
				return m, nil
			}
			m.done = true
			return m, tea.Quit

		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.inputs[m.focused].PromptStyle = lipgloss.NewStyle().Foreground(ColorMuted)
			m.focused = (m.focused + 1) % fieldCount
			m.inputs[m.focused].Focus()
			m.inputs[m.focused].PromptStyle = Prompt
			return m, textinput.Blink

		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.inputs[m.focused].PromptStyle = lipgloss.NewStyle().Foreground(ColorMuted)
			m.focused = (m.focused + fieldCount - 1) % fieldCount
			m.inputs[m.focused].Focus()
			m.inputs[m.focused].PromptStyle = Prompt
			return m, textinput.Blink

		case "enter":
			if m.focused < fieldNotes {
				// Advance to next field.
				m.inputs[m.focused].Blur()
				m.inputs[m.focused].PromptStyle = lipgloss.NewStyle().Foreground(ColorMuted)
				m.focused++
				m.inputs[m.focused].Focus()
				m.inputs[m.focused].PromptStyle = Prompt
				return m, textinput.Blink
			}
			// Last field: save.
			if m.inputs[fieldPassword].Value() == "" {
				m.err = "password cannot be empty"
				m.focused = fieldPassword
				m.inputs[fieldUsername].Blur()
				m.inputs[fieldNotes].Blur()
				m.inputs[fieldPassword].Focus()
				m.inputs[fieldPassword].PromptStyle = Prompt
				return m, textinput.Blink
			}
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

var activeFieldBox = lipgloss.NewStyle().
	BorderLeft(true).
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(ColorPrimary).
	PaddingLeft(1)

var inactiveFieldBox = lipgloss.NewStyle().
	BorderLeft(true).
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(ColorMuted).
	PaddingLeft(1)

func (m entryFormModel) View() string {
	var b strings.Builder
	b.WriteString(BigLogo())
	b.WriteByte('\n')

	action := "new entry"
	if m.inputs[fieldUsername].Value() != "" || m.inputs[fieldPassword].Value() != "" {
		action = "edit entry"
	}
	b.WriteString("  " + Title.Render(action+":") + " " + Key.Render(m.label) + "\n\n")

	labels := []string{"username", "password", "url     ", "notes   "}
	for i, inp := range m.inputs {
		lbl := Muted.Render(labels[i])
		if formField(i) == m.focused {
			b.WriteString(activeFieldBox.Render(lbl + "\n" + inp.View()))
		} else {
			b.WriteString(inactiveFieldBox.Render(lbl + "\n" + inp.View()))
		}
		b.WriteByte('\n')
	}

	if m.err != "" {
		b.WriteString("\n" + Errored.Render("✗ "+m.err) + "\n")
	}

	b.WriteString("\n" + HelpBar.Render(
		"tab/↓ next field  ·  shift+tab/↑ prev  ·  ctrl+s save  ·  esc cancel",
	))
	return b.String()
}

// ShowEntryForm launches the interactive add/edit form and returns the
// result. prefill can be zero-value for a new entry.
// Returns ErrPromptAborted if the user presses esc.
func ShowEntryForm(label string, prefill EntryForm) (EntryForm, error) {
	m := newEntryFormModel(label, prefill)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(os.Stdin))
	final, err := p.Run()
	if err != nil {
		return EntryForm{}, fmt.Errorf("form: %w", err)
	}
	fm := final.(entryFormModel)
	if fm.cancel {
		return EntryForm{}, ErrPromptAborted
	}
	return EntryForm{
		Label:    label,
		Username: fm.inputs[fieldUsername].Value(),
		Password: fm.inputs[fieldPassword].Value(),
		URL:      fm.inputs[fieldURL].Value(),
		Notes:    fm.inputs[fieldNotes].Value(),
	}, nil
}

// RenderEntryCard returns a styled card for one decrypted entry.
func RenderEntryCard(label, username, password, url, notes, updated string, showPassword bool) string {
	pw := strings.Repeat("•", min(len(password), 20))
	if showPassword {
		pw = password
	}

	rows := []struct{ k, v string }{
		{"username", username},
		{"password", pw},
	}
	if url != "" {
		rows = append(rows, struct{ k, v string }{"url", url})
	}
	if notes != "" {
		rows = append(rows, struct{ k, v string }{"notes", notes})
	}
	rows = append(rows, struct{ k, v string }{"updated", updated})

	kw := 0
	for _, r := range rows {
		if len(r.k) > kw {
			kw = len(r.k)
		}
	}

	var body strings.Builder
	body.WriteString(Key.Render(label) + "\n\n")
	for _, r := range rows {
		key := Muted.Render(pad(r.k, kw))
		val := r.v
		if r.k == "password" {
			val = lipgloss.NewStyle().Foreground(ColorAccent).Render(pw)
		}
		body.WriteString(key + "  " + val + "\n")
	}
	if !showPassword {
		body.WriteString("\n" + Muted.Render("use --show to reveal the password"))
	}

	return Box.Render(body.String())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
