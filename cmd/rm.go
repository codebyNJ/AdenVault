package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

var rmCmd = &cobra.Command{
	Use:     "rm <label>",
	Aliases: []string{"remove", "delete", "del", "d"},
	Short:   "delete an entry from the vault",
	Args:    cobra.ExactArgs(1),
	RunE:    runRm,
}

func runRm(cmd *cobra.Command, args []string) error {
	label := args[0]
	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}

	if ui.IsInteractive() && !flagPasswordStdin {
		ok, err := ui.Confirm(fmt.Sprintf("permanently delete %q from %s vault?", label, flagEnv), false)
		if err != nil {
			return formatError(err)
		}
		if !ok {
			info("cancelled — nothing changed")
			return nil
		}
	}

	if err := v.Delete(label); err != nil {
		if errors.Is(err, vault.ErrEntryNotFound) {
			return fmt.Errorf("entry not found: %s", label)
		}
		return err
	}
	if err := v.Save(); err != nil {
		return err
	}
	success(fmt.Sprintf("%s deleted", label))
	return nil
}
