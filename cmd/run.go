package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run -- <command> [args...]",
	Aliases: []string{"r", "exec", "x"},
	Short:   "run a command with vault entries injected as env vars",
	Long: `Decrypt all entries and inject them as environment variables
into a subprocess. Each entry becomes LABEL=password in the env.
Use --with-user to also inject LABEL_USER=username.

  adenV run -- npm start
  adenV --env prod run -- ./deploy.sh`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE:               runRun,
}

var flagRunWithUser bool

func init() {
	runCmd.Flags().SetInterspersed(false)
	runCmd.Flags().BoolVar(&flagRunWithUser, "with-user", false, "also inject LABEL_USER=username vars")
}

func runRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adenV run -- <command> [args...]")
	}

	v, err := loadVault()
	if err != nil {
		return formatError(err)
	}
	entries, err := v.All()
	if err != nil {
		return err
	}

	env := os.Environ()
	for _, e := range entries {
		key := strings.ToUpper(strings.ReplaceAll(e.Label, "-", "_"))
		env = append(env, fmt.Sprintf("%s=%s", key, e.Password))
		if flagRunWithUser && e.Username != "" {
			env = append(env, fmt.Sprintf("%s_USER=%s", key, e.Username))
		}
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env

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
