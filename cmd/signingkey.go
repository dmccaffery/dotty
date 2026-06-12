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

// SigningKeyFlags holds the flags shared by the signing-key verbs, persistent
// on the noun so they parse before or after the verb.
type SigningKeyFlags struct {
	SecurityKey string // serial number or alias
	Username    string
}

var signingKeyFlags = SigningKeyFlags{}

var signingKeyCmd = &cobra.Command{
	Use:     "signing-key <verb>",
	Aliases: []string{"ssh-key"},
	Short:   "Create and use SSH signing keys on hardware security keys.",
	Long: `Signing keys are resident FIDO2 credentials on a YubiKey, used to sign
git commits, tags, and files via ssh-keygen. dotty keeps only key-handle
stubs on disk (under the private $XDG_DATA_HOME/dotty/security-key) — the
secret never leaves the hardware. Keys are PIN-protected (verify-required)
and need no touch per signature.`,
	Example: `  dotty signing-key new
  dotty signing-key list
  dotty signing-key get --security-key=work
  dotty signing-key sign --print-git-config`,
}

func init() {
	signingKeyCmd.PersistentFlags().StringVar(&signingKeyFlags.SecurityKey, "security-key", "",
		"security key to use: a serial number or an alias")
	signingKeyCmd.PersistentFlags().StringVar(&signingKeyFlags.Username, "username", "",
		"username the key is enrolled under (default: the current user)")
	rootCmd.AddCommand(signingKeyCmd)
}
