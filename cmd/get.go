package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"aden/internal/vault"
)

var getCmd = &cobra.Command{
	Use:     "get <KEY>",
	Aliases: []string{"g", "show", "cat", "read"},
	Short:   "print a decrypted secret to stdout",
	Long: `Print a decrypted secret value to stdout.

Output is the raw value only — no labels, no colours — so it is
safe to use in subshells: aden get DB_URL | psql, $(aden get TOKEN), etc.`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}
	value, err := v.Get(name)
	if err != nil {
		if errors.Is(err, vault.ErrKeyNotFound) {
			return fmt.Errorf("key not found: %s", name)
		}
		return err
	}
	// Print plain value to stdout — never decorate this output.
	fmt.Fprint(cmd.OutOrStdout(), value)
	// Trailing newline is convenient for humans; subshells trim it.
	fmt.Fprintln(cmd.OutOrStdout())
	return nil
}
