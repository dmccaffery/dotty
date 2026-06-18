// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/profile"
)

// brewfileCmd groups the brewfile verbs.
var brewfileCmd = &cobra.Command{
	Use:     "brewfile <verb>",
	Aliases: []string{"brew"},
	Short:   "Manage the profile's Brewfile for reproducible brews.",
	Long: `Maintain a homebrew bundle Brewfile so a machine's brews stay reproducible
on and across systems. Commands operate on the active profile's Brewfile, or
on a specific profile's via the global --profile flag.`,
	Example: `  dotty brewfile add ripgrep
  dotty brewfile add --cask ghostty
  dotty --profile=work brewfile sync
  dotty brew upgrade`,
}

func init() {
	rootCmd.AddCommand(brewfileCmd)
}

// resolveBrewfilePath finds the Brewfile the brewfile verbs operate on: the
// --profile flag's profile when given (it must exist), otherwise the active
// profile.
func resolveBrewfilePath() (string, error) {
	configDir, err := cli.ConfigDir()
	if err != nil {
		return "", err
	}
	if rootFlags.Profile != "" {
		if !profile.Exists(configDir, rootFlags.Profile) {
			return "", fmt.Errorf("profile %q: %w", rootFlags.Profile, profile.ErrNotFound)
		}
		return profile.BrewfilePath(profile.Dir(configDir, rootFlags.Profile)), nil
	}
	dir, err := profile.ActiveDir(configDir)
	if errors.Is(err, profile.ErrNoActiveProfile) {
		return "", fmt.Errorf("%w, or pass --profile", err)
	}
	if err != nil {
		return "", err
	}
	return profile.BrewfilePath(dir), nil
}
