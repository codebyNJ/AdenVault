package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type confirmModel struct {
	prompt   string
	selected int // 0 = yes, 1 = no
	answered bool
	cancel   bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit
		case "left", "right", "h", "l", "tab":
			m.selected = 1 - m.selected
		case "y", "Y":
			m.selected = 0
			m.answered = true
			return m, tea.Quit
		case "n", "N":
			m.selected = 1
			m.answered = true
			return m, tea.Quit
		case "enter":
			m.answered = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	yes := "  yes  "
	no := "  no  "
	if m.selected == 0 {
		yes = Prompt.Render("[ yes ]")
		no = Muted.Render("  no  ")
	} else {
		yes = Muted.Render("  yes ")
		no = Prompt.Render("[ no ]")
	}
	var b strings.Builder
	b.WriteString(Title.Render(m.prompt) + "\n\n")
	b.WriteString(yes + "   " + no + "\n\n")
	b.WriteString(HelpBar.Render("←/→ choose  •  y/n shortcut  •  enter confirm"))
	return b.String()
}

// Confirm shows an interactive yes/no prompt and returns true if the
// user accepts. defaultYes controls the highlighted button on entry.
//
// In non-interactive sessions it returns defaultYes without prompting,
// so piped invocations remain non-blocking.
func Confirm(prompt string, defaultYes bool) (bool, error) {
	if !isInteractive() {
		return defaultYes, nil
	}
	m := confirmModel{prompt: prompt}
	if !defaultYes {
		m.selected = 1
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(os.Stdin))
	final, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("confirm: %w", err)
	}
	fm := final.(confirmModel)
	if fm.cancel {
		return false, ErrPromptAborted
	}
	return fm.selected == 0, nil
}
