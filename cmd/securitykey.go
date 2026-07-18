// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/profile"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
)

// SecurityKeyFlags holds the flags shared by the security-key verbs.
type SecurityKeyFlags struct {
	// Serial is persistent on the noun so it parses in both positions:
	// `dotty security-key --serial=N add` and `dotty security-key add --serial=N`.
	Serial string
}

var securityKeyFlags = SecurityKeyFlags{}

var securityKeyCmd = &cobra.Command{
	Use:     "security-key <verb>",
	Aliases: []string{"sk"},
	Short:   "Manage hardware security keys.",
	Long: `Name your hardware security keys: aliases map memorable names to YubiKey
serial numbers, so other commands can say --security-key=work instead of a
serial. Aliases live in the private dotty data directory
($XDG_DATA_HOME/dotty/security-key), not in shareable config.`,
	Example: `  dotty security-key add --name=work
  dotty sk --serial=12345678 add --name=backup
  dotty security-key remove`,
}

func init() {
	securityKeyCmd.PersistentFlags().StringVar(&securityKeyFlags.Serial, "serial", "",
		"serial number of the security key")
	rootCmd.AddCommand(securityKeyCmd)
}

// keyProfileDir resolves the profile whose security-key state (aliases,
// allowlist) a command operates on: --profile, else the active profile. The
// profile must exist — its directory holds the store.
func keyProfileDir() (string, error) {
	configDir, err := cli.ConfigDir()
	if err != nil {
		return "", err
	}
	name := rootFlags.Profile
	if name == "" {
		if name, err = profile.ActiveName(configDir); err != nil {
			return "", fmt.Errorf("no active profile; pass --profile or run dotty init: %w", err)
		}
	}
	if !profile.Exists(configDir, name) {
		return "", fmt.Errorf("profile %q: %w", name, profile.ErrNotFound)
	}
	return profile.Dir(configDir, name), nil
}

// keyStore loads the profile's security-key alias store.
func keyStore() (*securitykey.Store, error) {
	dir, err := keyProfileDir()
	if err != nil {
		return nil, err
	}
	return securitykey.LoadStore(securitykey.StorePath(dir))
}
