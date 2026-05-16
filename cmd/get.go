package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

var flagShow bool

var getCmd = &cobra.Command{
	Use:     "get <label>",
	Aliases: []string{"g", "show", "view", "open"},
	Short:   "view an entry from the vault",
	Long: `Decrypt and display one entry from the vault.

The password is masked by default. Use --show to reveal it:

  adenV get github --show

To copy the password to your clipboard without displaying it, use:

  adenV copy github`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	getCmd.Flags().BoolVar(&flagShow, "show", false, "reveal the password in the output")
}

func runGet(cmd *cobra.Command, args []string) error {
	label := args[0]
	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}
	entry, err := v.Get(label)
	if err != nil {
		if errors.Is(err, vault.ErrEntryNotFound) {
			return fmt.Errorf("entry not found: %s", label)
		}
		return err
	}

	updated := entry.UpdatedAt.Local().Format("2006-01-02 15:04")
	card := ui.RenderEntryCard(
		entry.Label, entry.Username, entry.Password,
		entry.URL, entry.Notes, updated, flagShow,
	)
	fmt.Fprintln(os.Stderr, card)
	return nil
}
