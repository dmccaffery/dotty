// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/brewfile"
	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

var brewfileUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade everything in the Brewfile.",
	Long: `Install and upgrade all brews in the Brewfile without removing anything —
brew bundle install --upgrade.`,
	Example: `  dotty brewfile upgrade
  dotty --profile=work brewfile upgrade`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		path, err := resolveBrewfilePath()
		if err != nil {
			return err
		}
		if err := brewfile.Upgrade(cmd.Context(), newRunner(ios), path); err != nil {
			return err
		}
		tui.Successf(ios, "Upgraded brews from %s", path)
		return nil
	},
}

func init() {
	brewfileCmd.AddCommand(brewfileUpgradeCmd)
}
