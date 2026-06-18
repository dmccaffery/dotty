// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"github.com/spf13/cobra"
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
