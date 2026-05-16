package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:     "set <KEY> <VALUE>",
	Aliases: []string{"s", "add", "put", "save"},
	Short:   "store an encrypted secret",
	Long: `Store an encrypted secret in the current project's vault.

The value is AES-256-GCM encrypted with a key derived from your
master password before being written to disk.`,
	Args: cobra.ExactArgs(2),
	RunE: runSet,
}

func runSet(cmd *cobra.Command, args []string) error {
	name, value := args[0], args[1]

	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}
	if err := v.Set(name, value); err != nil {
		return err
	}
	if err := v.Save(); err != nil {
		return err
	}
	success(fmt.Sprintf("%s saved", name))
	return nil
}
