package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i", "new", "create"},
	Short:   "create a new encrypted vault for this project",
	Long: `Create a new encrypted vault for the current project.

The vault lives at ~/.aden/<project>-<id>/vault.<env>.json. You will
be prompted for a master password (twice). The password is never
written to disk — only a salt and the encrypted values are persisted.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	p, err := resolveProject()
	if err != nil {
		return err
	}
	path := p.VaultPath(flagEnv)

	// Splash on interactive sessions only — non-TTY runs (CI) get a
	// quiet, terse flow.
	if ui.IsInteractive() && !flagQuiet {
		printSplash()
	}

	if vault.Exists(path) {
		warn(fmt.Sprintf("a %s vault already exists at %s", flagEnv, path))
		// Without a TTY we can't ask for confirmation. Refuse to
		// silently overwrite — the caller can delete the file
		// explicitly if they really want to start fresh.
		if !ui.IsInteractive() || flagPasswordStdin {
			return fmt.Errorf("refusing to overwrite existing vault non-interactively; delete it first if intentional")
		}
		ok, err := ui.Confirm("overwrite existing vault? this destroys all secrets in it.", false)
		if err != nil {
			return formatError(err)
		}
		if !ok {
			info("init cancelled — existing vault untouched")
			return nil
		}
	}

	pw, err := promptPassword(fmt.Sprintf("set master password for %s vault", flagEnv), true)
	if err != nil {
		return formatError(err)
	}

	v, err := vault.New(path, p.Name, flagEnv, pw)
	if err != nil {
		return err
	}
	if err := v.Save(); err != nil {
		return err
	}
	if err := vault.SaveConfig(p); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, ui.Box.Render(fmt.Sprintf(
		"%s vault ready\n\n%s %s\n%s %s\n%s %s",
		ui.Success.Render("✓"),
		ui.Muted.Render("project:"), ui.Key.Render(p.Name),
		ui.Muted.Render("env:    "), ui.Key.Render(flagEnv),
		ui.Muted.Render("path:   "), ui.Muted.Render(path),
	)))
	fmt.Fprintln(os.Stderr)
	info("next: store your first secret with " + ui.Key.Render("adenV set DB_URL postgres://..."))
	return nil
}
