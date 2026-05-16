package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"e", "dump", "env"},
	Short:   "export all entries as LABEL=password lines",
	Long: `Decrypt every entry and print LABEL=password lines to stdout.

  adenV export > .env

Note: only the password field is exported per line. For username use
--with-user to get LABEL_USER=username lines alongside.`,
	Args: cobra.NoArgs,
	RunE: runExport,
}

var flagWithUser bool

func init() {
	exportCmd.Flags().BoolVar(&flagWithUser, "with-user", false, "also export LABEL_USER=username lines")
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
		key := strings.ToUpper(strings.ReplaceAll(e.Label, "-", "_"))
		fmt.Fprintf(out, "%s=%s\n", key, shellEscape(e.Password))
		if flagWithUser && e.Username != "" {
			fmt.Fprintf(out, "%s_USER=%s\n", key, shellEscape(e.Username))
		}
	}
	if !flagQuiet {
		warn("remember to add .env to your .gitignore")
	}
	return nil
}

func shellEscape(v string) string {
	if !needsQuoting(v) {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + r.Replace(v) + `"`
}

func needsQuoting(v string) bool {
	if v == "" {
		return true
	}
	for _, c := range v {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '_', c == '-', c == '.', c == '/', c == ':', c == '+':
		default:
			return true
		}
	}
	return false
}
