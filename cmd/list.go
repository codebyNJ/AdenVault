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
	Aliases: []string{"ls", "l", "all"},
	Short:   "list all entries in the vault (no password required)",
	Long: `List every entry label in the vault along with its last-updated
timestamp. Entry labels are stored in plaintext so this command
never asks for your master password.`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	p, err := resolveProject()
	if err != nil {
		return err
	}
	v, err := vault.LoadEncrypted(p.VaultPath(flagEnv))
	if err != nil {
		return formatError(err)
	}

	metas := v.Metas()

	if flagQuiet {
		for _, m := range metas {
			fmt.Fprintln(cmd.OutOrStdout(), m.Label)
		}
		return nil
	}

	rows := make([]ui.EntryMeta, 0, len(metas))
	for _, m := range metas {
		rows = append(rows, ui.EntryMeta{
			Label:     m.Label,
			HasUser:   m.HasUser,
			HasURL:    m.HasURL,
			HasNotes:  m.HasNotes,
			UpdatedAt: m.UpdatedAt,
		})
	}
	fmt.Fprintln(os.Stderr, ui.RenderList(v.Project(), v.Environment(), rows))
	return nil
}
