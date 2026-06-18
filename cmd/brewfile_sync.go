// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/brewfile"
	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// BrewfileSyncFlags holds the flags for `dotty brewfile sync`.
type BrewfileSyncFlags struct {
	Force bool
}

var brewfileSyncFlags = BrewfileSyncFlags{}

var brewfileSyncCmd = &cobra.Command{
	Use:   "sync [--force]",
	Short: "Make the machine match the Brewfile exactly.",
	Long: `Synchronise the machine with the Brewfile — install what's listed, upgrade
what's outdated, and remove (zap) what isn't listed. When anything would be
removed, dotty shows the list and asks first unless --force is set.`,
	Example: `  dotty brewfile sync
  dotty brewfile sync --force`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		path, err := resolveBrewfilePath()
		if err != nil {
			return err
		}
		aborted := false
		confirm := func(removals []string) (bool, error) {
			tui.Warnf(ios, "Syncing will remove brews not in the Brewfile:")
			for _, line := range removals {
				_, _ = fmt.Fprintf(ios.ErrOut, "    %s\n", line)
			}
			ok, err := tui.Confirm(ios, "Remove them and continue?", "")
			if errors.Is(err, tui.ErrNotInteractive) {
				return false, errors.New("sync would remove brews; re-run interactively or pass --force")
			}
			if errors.Is(err, tui.ErrAborted) {
				err = nil
			}
			aborted = !ok
			return ok, err
		}
		if err := brewfile.Sync(cmd.Context(), newRunner(ios), path, brewfileSyncFlags.Force, confirm); err != nil {
			return err
		}
		if aborted {
			tui.Infof(ios, "Sync aborted; nothing changed")
			return nil
		}
		tui.Successf(ios, "Machine synced with %s", path)
		return nil
	},
}

func init() {
	brewfileSyncCmd.Flags().BoolVar(&brewfileSyncFlags.Force, "force", false,
		"remove unlisted brews without asking")
	brewfileCmd.AddCommand(brewfileSyncCmd)
}
