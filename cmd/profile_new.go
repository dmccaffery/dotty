// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/profile"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// ProfileNewFlags holds the flags for `dotty profile new`.
type ProfileNewFlags struct {
	Name        string
	Description string
	Activate    bool
}

var profileNewFlags = ProfileNewFlags{}

var profileNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a system-level profile.",
	Long: `Create a profile directory under $XDG_CONFIG_HOME/dotty/<name>. Without
--name dotty prompts for one. Unless --activate is given, dotty asks whether
to activate the new profile right away.`,
	Example: `  dotty profile new
  dotty profile new --name=work --description="work laptop" --activate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createProfile(cmd.Context(), cli.System(),
			profileNewFlags.Name, profileNewFlags.Description, profileNewFlags.Activate)
	},
}

func init() {
	profileNewCmd.Flags().StringVar(&profileNewFlags.Name, "name", "", "name for the new profile")
	profileNewCmd.Flags().StringVar(&profileNewFlags.Description, "description", "", "short description of the profile")
	profileNewCmd.Flags().BoolVar(&profileNewFlags.Activate, "activate", false, "activate the profile after creating it")
	profileCmd.AddCommand(profileNewCmd)
}

// createProfile is the full `profile new` flow. The activate command's
// create-when-missing path calls it too — the shared-function equivalent of
// invoking `dotty profile new --name=<name> --activate`.
func createProfile(ctx context.Context, ios cli.IOStreams, name, description string, activate bool) error {
	configDir, err := cli.ConfigDir()
	if err != nil {
		return err
	}

	if name == "" {
		name, err = tui.Input(ios, "Profile name", "work", func(s string) error {
			if err := profile.ValidateName(s); err != nil {
				return err
			}
			if profile.Exists(configDir, s) {
				return fmt.Errorf("profile %q already exists", s)
			}
			return nil
		})
		if errors.Is(err, tui.ErrNotInteractive) {
			return errors.New("no profile name given; pass --name or run interactively")
		}
		if err != nil {
			return err
		}
	}

	p, err := profile.Create(configDir, name, description)
	if err != nil {
		return err
	}
	tui.Successf(ios, "Created profile %s (%s)", p.Name, profile.Dir(configDir, p.Name))

	if !activate && ios.IsInteractive() {
		ok, err := tui.Confirm(ios, fmt.Sprintf("Activate profile %q now?", p.Name), "")
		if err != nil && !errors.Is(err, tui.ErrAborted) {
			return err
		}
		activate = ok
	}
	if !activate {
		return nil
	}
	return activateProfile(ctx, ios, configDir, p.Name)
}
