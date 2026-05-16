package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l", "keys"},
	Short:   "list secret names (no password required)",
	Long: `List all secret names in the current project's vault.

Names are stored in plaintext so this command never asks for the
master password — useful for quickly checking what's stored.`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	p, err := resolveProject()
	if err != nil {
		return err
	}
	path := p.VaultPath(flagEnv)
	v, err := vault.LoadEncrypted(path)
	if err != nil {
		return formatError(err)
	}

	rows := make([]ui.SecretRow, 0, v.Count())
	for _, e := range v.Metadata() {
		rows = append(rows, ui.SecretRow{Name: e.Name, UpdatedAt: e.UpdatedAt})
	}

	out := os.Stderr
	if flagQuiet {
		// In --quiet mode, dump plain names only to stdout for scripting.
		for _, r := range rows {
			fmt.Fprintln(cmd.OutOrStdout(), r.Name)
		}
		return nil
	}
	fmt.Fprintln(out, ui.RenderList(v.Project(), v.Environment(), rows))
	return nil
}
