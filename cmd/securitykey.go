// MIT License
//
// Copyright (c) 2026 Bitwise Media Group
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
