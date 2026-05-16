package cmd

import (
	"errors"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"aden/internal/vault"
)

var flagCopyUser bool

var copyCmd = &cobra.Command{
	Use:     "copy <label>",
	Aliases: []string{"cp", "c", "yank"},
	Short:   "copy a password (or username) to the clipboard",
	Long: `Decrypt an entry and copy the password to the clipboard
without printing it to the terminal.

  adenV copy github          # copies password
  adenV copy github --user   # copies the username instead`,
	Args: cobra.ExactArgs(1),
	RunE: runCopy,
}

func init() {
	copyCmd.Flags().BoolVarP(&flagCopyUser, "user", "u", false, "copy the username instead of the password")
}

func runCopy(cmd *cobra.Command, args []string) error {
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

	value := entry.Password
	field := "password"
	if flagCopyUser {
		value = entry.Username
		field = "username"
	}
	if value == "" {
		return fmt.Errorf("%s has no %s stored", label, field)
	}

	if err := clipboard.WriteAll(value); err != nil {
		return fmt.Errorf("clipboard: %w\ntip: use `adenV get %s --show` to reveal it manually", err, label)
	}

	success(fmt.Sprintf("%s %s copied to clipboard", label, field))
	return nil
}
