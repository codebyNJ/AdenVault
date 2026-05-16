package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"e", "dump", "env"},
	Short:   "print all secrets as KEY=value lines on stdout",
	Long: `Decrypt every secret in the current project's vault and print
them as KEY=value lines on stdout — one per secret. Designed to be
redirected:

  aden export > .env

Remember to add .env to your .gitignore.`,
	Args: cobra.NoArgs,
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}
	entries, err := v.All()
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, e := range entries {
		fmt.Fprintf(out, "%s=%s\n", e.Name, shellEscape(e.Value))
	}
	if !flagQuiet {
		warn("remember to add .env to your .gitignore before committing")
	}
	return nil
}

// shellEscape returns a value safe to use on the right-hand side of a
// KEY=value line. Values without whitespace, quotes, or shell
// metacharacters pass through untouched; anything else is wrapped in
// double quotes with embedded "/$/\/` escaped.
func shellEscape(v string) string {
	if !needsQuoting(v) {
		return v
	}
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"`", "\\`",
		`$`, `\$`,
	)
	return `"` + r.Replace(v) + `"`
}

func needsQuoting(v string) bool {
	if v == "" {
		return true
	}
	for _, c := range v {
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '_', c == '-', c == '.', c == '/', c == ':', c == '+':
			// safe
		default:
			return true
		}
	}
	return false
}
