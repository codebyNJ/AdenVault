// Package cmd wires the cobra CLI on top of the vault/crypto/ui
// packages. Every user-facing command lives in this directory.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"aden/internal/ui"
	"aden/internal/vault"
)

// Version is set at link time by `go build -ldflags "-X aden/cmd.Version=..."`.
var Version = "dev"

// global flag state — populated by cobra before subcommand RunE fires.
var (
	flagEnv          string
	flagVaultDir     string
	flagPasswordStdin bool
	flagQuiet        bool
)

var rootCmd = &cobra.Command{
	Use:           "adenV",
	Short:         "adenVault — your offline password manager. no cloud. no subscription.",
	Long:          "adenVault is a personal CLI password manager.\nStore passwords, usernames, and notes — encrypted with AES-256-GCM,\nlocked behind your master password (Argon2id), never leaves your machine.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
	// Bare `aden` shows the gradient splash with project status. Users
	// can still hit `aden --help` for the full command list (cobra
	// handles --help before Run fires).
	Run: func(cmd *cobra.Command, args []string) {
		printSplash()
	},
}

// printSplash writes the big gradient banner to stderr. If the current
// directory already has a vault, the status line shows the project /
// env / secret count — same shape as the reference screenshot.
func printSplash() {
	info := ui.SplashInfo{Env: flagEnv}
	if p, err := resolveProject(); err == nil {
		path := p.VaultPath(flagEnv)
		if vault.Exists(path) {
			if v, err := vault.LoadEncrypted(path); err == nil {
				info.Project = v.Project()
				info.Env = v.Environment()
				info.Count = v.Count()
				info.Exists = true
			}
		}
	}
	fmt.Fprintln(os.Stderr, ui.Splash(info))
}

// Execute runs the root command. main wraps the returned error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagEnv, "env", "dev", "environment name (dev/staging/prod/...)")
	rootCmd.PersistentFlags().StringVar(&flagVaultDir, "vault-dir", "", "override the default vault directory (~/.aden)")
	rootCmd.PersistentFlags().BoolVar(&flagPasswordStdin, "password-stdin", false, "read master password from stdin (CI / non-TTY)")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "suppress status output")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(runCmd)
}

// resolveProject returns the vault project for the current working
// directory, honouring --vault-dir if set.
func resolveProject() (vault.Project, error) {
	return vault.ResolveProject(flagVaultDir)
}

// promptPassword prompts the user (or reads stdin) and returns the
// password as a byte slice.
func promptPassword(label string, confirm bool) ([]byte, error) {
	if flagPasswordStdin {
		return ui.PromptPasswordStdin()
	}
	return ui.PromptPassword(label, confirm, false)
}

// loadVault opens the current vault (must exist) and prompts for the
// password. Returns wrapped errors with recovery hints.
func loadVault() (*vault.Vault, error) {
	p, err := resolveProject()
	if err != nil {
		return nil, err
	}
	path := p.VaultPath(flagEnv)
	if !vault.Exists(path) {
		return nil, fmt.Errorf("%w (env=%s)", vault.ErrVaultNotFound, flagEnv)
	}
	pw, err := promptPassword(fmt.Sprintf("unlock %s vault", flagEnv), false)
	if err != nil {
		return nil, err
	}
	v, err := vault.Load(path, pw)
	if err != nil {
		return nil, err
	}
	if err := v.VerifyPassword(); err != nil {
		return nil, err
	}
	return v, nil
}

// info prints a styled message to stderr (so stdout stays clean for
// piping), unless --quiet was passed.
func info(msg string) {
	if flagQuiet {
		return
	}
	ui.PrintInfo(os.Stderr, msg)
}

// success prints a green check + message to stderr.
func success(msg string) {
	if flagQuiet {
		return
	}
	ui.PrintSuccess(os.Stderr, msg)
}

// warn prints a yellow warning to stderr.
func warn(msg string) {
	if flagQuiet {
		return
	}
	ui.PrintWarn(os.Stderr, msg)
}

// formatError wraps low-level errors with a friendly recovery hint so
// users don't see raw stack-trace-ish output.
func formatError(err error) error {
	switch {
	case errors.Is(err, vault.ErrVaultNotFound):
		return fmt.Errorf("vault not found — run `adenV init` first")
	case errors.Is(err, vault.ErrEntryNotFound):
		return err
	case errors.Is(err, ui.ErrPromptAborted):
		return errors.New("aborted")
	}
	return err
}
