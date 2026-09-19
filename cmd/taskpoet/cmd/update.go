package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/drewstinnett/taskpoet/v2/internal/update"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	// updateWait is the most a finished command holds up the prompt for the
	// lazy update check, which is only ever due once every six hours
	updateWait = time.Second

	checkTimeout  = 10 * time.Second
	updateTimeout = 2 * time.Minute
)

// lazyUpdate is the update check that is running along with the command, if any
var lazyUpdate *update.Lazy

// commands that must never print an update notice, or wait for the network
var noUpdateCheck = map[string]bool{
	"update":                        true,
	"help":                          true,
	"completion":                    true,
	cobra.ShellCompRequestCmd:       true,
	cobra.ShellCompNoDescRequestCmd: true,
}

// startUpdateCheck is run before every command. It looks at the notes from the
// last check, and if that was over six hours ago it asks GitHub in the
// background. Nothing here can fail a command.
func startUpdateCheck(cmd *cobra.Command, _ []string) {
	if !viper.GetBool("update.check") || os.Getenv("CI") != "" {
		return
	}
	// A notice is for a person at a terminal. Scripts, cron and pipes get none,
	// and don't spend time on the network either.
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}
	for c := cmd; c != nil; c = c.Parent() {
		if noUpdateCheck[c.Name()] {
			return
		}
	}
	cur, err := update.ParseVersion(version)
	if err != nil {
		return // a development build has no idea what is newer
	}
	checker, err := update.NewChecker(update.NewSource(version), cur)
	if err != nil {
		return
	}
	lazyUpdate = checker.Start()
}

// announceUpdate says so if the check found a newer version. Once found, it
// says so at most every six hours, not on every command.
func announceUpdate() {
	v, ok := lazyUpdate.Wait(updateWait)
	if !ok {
		return
	}
	fmt.Fprintf(os.Stderr, "\nA new version of taskpoet is available: %s (this is %s)\n%s\n", v.Tag(), version, updateHint())
}

func updateHint() string {
	if exe, err := update.Executable(); err == nil && update.ManagedBy(exe) == "Homebrew" {
		return "Update it with: brew upgrade taskpoet"
	}
	return "Update it with: taskpoet update"
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update taskpoet to the latest release",
		Long: `Download the latest release from GitHub, check it against the checksums that are
published with it, and replace this program with it.

taskpoet also checks for new releases by itself, at most every six hours and
only when run from a terminal. It tells you when there is one, and never
installs anything without being asked. Turn that off with 'update: {check: false}'
in the config file, or with TASKPOET_UPDATE_CHECK=false.

If taskpoet came from Homebrew, use 'brew upgrade taskpoet' instead.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: noComplete,
		RunE:              runUpdate,
	}
	cmd.Flags().Bool("check", false, "Only say whether there is a newer release, don't install it")
	return cmd
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	cur, err := update.ParseVersion(version)
	if err != nil {
		return fmt.Errorf("this is a development build (%s), only a release can update itself", version)
	}
	src := update.NewSource(version)

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	latest, err := src.Latest(ctx)
	cancel()
	if err != nil {
		return fmt.Errorf("looking for the latest release: %w", err)
	}
	if !latest.After(cur) {
		fmt.Printf("taskpoet %s is the latest release\n", cur.Tag())
		return nil
	}
	if mustGetCmd[bool](cmd, "check") {
		fmt.Printf("taskpoet %s is available, this is %s. Run 'taskpoet update' to install it.\n", latest.Tag(), cur.Tag())
		return nil
	}

	exe, err := update.Executable()
	if err != nil {
		return err
	}
	if by := update.ManagedBy(exe); by != "" {
		return fmt.Errorf("%s belongs to %s, which should do the update: brew upgrade taskpoet", exe, by)
	}
	fmt.Printf("Updating %s from %s to %s\n", exe, cur.Tag(), latest.Tag())
	ctx, cancel = context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	if err := src.Install(ctx, latest, exe); err != nil {
		if update.IsPermission(err) {
			return fmt.Errorf("can't write to %s: %w\nRun the update again as a user that can, with sudo for instance", filepath.Dir(exe), err)
		}
		return fmt.Errorf("updating: %w", err)
	}
	fmt.Printf("Updated to %s\n", latest.Tag())
	return nil
}
