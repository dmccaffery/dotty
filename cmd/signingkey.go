// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"cmp"
	"slices"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/signingkey"
)

// preferDefaultKeyType sorts refs so ed25519 (the default key type) comes
// first, making single-key selection deterministic.
func preferDefaultKeyType(refs []signingkey.KeyRef) {
	slices.SortFunc(refs, func(a, b signingkey.KeyRef) int { return cmp.Compare(b.Type, a.Type) })
}

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
