// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import "github.com/spf13/cobra"

// profileCmd groups the profile verbs.
var profileCmd = &cobra.Command{
	Use:   "profile <verb>",
	Short: "Manage system profiles that travel across machines.",
	Long: `Profiles are per-machine configuration sets — a Brewfile today; prompt and
terminal themes later — stored under $XDG_CONFIG_HOME/dotty/<name> so a public
dotfiles repository can carry them. One profile is active at a time, named by
the active-profile symlink.`,
	Example: `  dotty profile new --name=work --description="work laptop"
  dotty profile activate
  dotty profile activate --name=personal`,
}

func init() {
	rootCmd.AddCommand(profileCmd)
}
