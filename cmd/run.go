package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run -- <command> [args...]",
	Aliases: []string{"r", "exec", "x"},
	Short:   "run a command with secrets injected as environment variables",
	Long: `Decrypt every secret for the current project + environment, merge
them with the existing shell environment (aden secrets win on conflict),
and exec the requested command. The subprocess's exit code is
forwarded to the shell.

No secrets are written to disk during this flow.

Example:

  aden run -- npm start
  aden --env prod run -- ./deploy.sh`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE:               runRun,
}

func init() {
	// We don't want cobra to interpret flags meant for the wrapped
	// command. Users invoke `aden run -- npm start --foo`; cobra
	// already understands the `--` separator and stops parsing there.
	runCmd.Flags().SetInterspersed(false)
}

func runRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aden run -- <command> [args...]")
	}

	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}
	entries, err := v.All()
	if err != nil {
		return err
	}

	// Build env: start with parent env, then overlay aden secrets.
	env := os.Environ()
	for _, e := range entries {
		env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env

	// Forward signals so ctrl-c reaches the child cleanly.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh)
	defer signal.Stop(sigCh)

	if err := c.Start(); err != nil {
		return fmt.Errorf("start %s: %w", args[0], err)
	}

	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				if c.Process != nil {
					_ = c.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	waitErr := c.Wait()
	close(done)

	if waitErr == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return waitErr
}
