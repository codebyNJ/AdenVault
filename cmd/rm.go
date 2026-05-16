package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

var rmCmd = &cobra.Command{
	Use:     "rm <KEY>",
	Aliases: []string{"remove", "delete", "del", "unset"},
	Short:   "delete a secret from the vault",
	Args:    cobra.ExactArgs(1),
	RunE:    runRm,
}

func runRm(cmd *cobra.Command, args []string) error {
	name := args[0]

	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}

	// In a real TTY, double-check before destroying a secret. In a
	// non-interactive session (--password-stdin, CI) the user has
	// explicitly opted into a no-prompt flow, so we proceed.
	if ui.IsInteractive() && !flagPasswordStdin {
		ok, err := ui.Confirm(fmt.Sprintf("delete %s from %s vault?", name, flagEnv), false)
		if err != nil {
			return formatError(err)
		}
		if !ok {
			info("cancelled — nothing changed")
			return nil
		}
	}

	if err := v.Delete(name); err != nil {
		if errors.Is(err, vault.ErrKeyNotFound) {
			return fmt.Errorf("key not found: %s", name)
		}
		return err
	}
	if err := v.Save(); err != nil {
		return err
	}
	success(fmt.Sprintf("%s removed", name))
	return nil
}
