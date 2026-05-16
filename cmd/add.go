package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

var (
	flagUser string
	flagPass string
	flagURL  string
	flagNote string
)

var addCmd = &cobra.Command{
	Use:     "add <label>",
	Aliases: []string{"a", "new", "create", "set", "save"},
	Short:   "add or update an entry in the vault",
	Long: `Add a new entry (or update an existing one) in the vault.

Run with just a label to get an interactive form:

  adenV add github

Or pass values directly for scripting:

  adenV add github --user john@example.com --pass mysecret --url https://github.com`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&flagUser, "user", "u", "", "username or email")
	addCmd.Flags().StringVarP(&flagPass, "pass", "p", "", "password (prompted if omitted)")
	addCmd.Flags().StringVar(&flagURL, "url", "", "website URL (optional)")
	addCmd.Flags().StringVarP(&flagNote, "note", "n", "", "notes (optional)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	label := args[0]

	// Need the vault open before prompting so we can pre-fill if the
	// entry already exists.
	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}

	// Pre-fill from existing entry if present (silent — we just use the
	// decrypted data as defaults for the form).
	prefill := ui.EntryForm{Label: label}
	if existing, err := v.Get(label); err == nil {
		prefill.Username = existing.Username
		prefill.Password = existing.Password
		prefill.URL = existing.URL
		prefill.Notes = existing.Notes
	}

	var form ui.EntryForm

	if flagPass != "" || flagPasswordStdin {
		// Non-interactive / scripted mode: use flags directly.
		form = ui.EntryForm{
			Label:    label,
			Username: cliOrDefault(flagUser, prefill.Username),
			Password: cliOrDefault(flagPass, prefill.Password),
			URL:      cliOrDefault(flagURL, prefill.URL),
			Notes:    cliOrDefault(flagNote, prefill.Notes),
		}
		if form.Password == "" {
			return fmt.Errorf("--pass is required in non-interactive mode")
		}
	} else {
		// Interactive Bubble Tea form.
		if flagUser != "" {
			prefill.Username = flagUser
		}
		if flagURL != "" {
			prefill.URL = flagURL
		}
		if flagNote != "" {
			prefill.Notes = flagNote
		}
		form, err = ui.ShowEntryForm(label, prefill)
		if err != nil {
			if errors.Is(err, ui.ErrPromptAborted) {
				info("cancelled — nothing changed")
				return nil
			}
			return err
		}
	}

	if err := v.Add(vault.EntryData{
		Label:    form.Label,
		Username: form.Username,
		Password: form.Password,
		URL:      form.URL,
		Notes:    form.Notes,
	}); err != nil {
		return err
	}
	if err := v.Save(); err != nil {
		return err
	}
	success(fmt.Sprintf("%s saved", label))
	return nil
}

func cliOrDefault(flag, def string) string {
	if flag != "" {
		return flag
	}
	return def
}
