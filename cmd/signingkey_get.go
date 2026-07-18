// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
)

// SigningKeyGetFlags holds the flags for `dotty signing-key get`.
type SigningKeyGetFlags struct {
	Format string
}

var signingKeyGetFlags = SigningKeyGetFlags{}

var signingKeyGetCmd = &cobra.Command{
	Use:   "get [--format=<text|key>]",
	Short: "Print a signing key's stub and public key.",
	Long: `Print the private key stub and public key for a username on a security
key. --format=key prints a single key::<public-key> line for git's
gpg.ssh.defaultKeyCommand; that mode never prompts (git captures the output),
preferring the ed25519 key when a user has several types enrolled.`,
	Example: `  dotty signing-key get
  dotty signing-key get --security-key=work --username=deavon
  dotty signing-key get --format=key   # for gpg.ssh.defaultKeyCommand`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		gitMode := signingKeyGetFlags.Format == "key"
		if !gitMode && signingKeyGetFlags.Format != "text" {
			return fmt.Errorf("unknown format %q (expected text or key)", signingKeyGetFlags.Format)
		}
		if gitMode {
			// git captures stdout/stderr — any prompt would hang it.
			ios = cli.IOStreams{In: nil, Out: ios.Out, ErrOut: ios.ErrOut}
		}
		dataDir, err := cli.DataDir()
		if err != nil {
			return err
		}
		store, err := keyStore()
		if err != nil {
			return err
		}

		serial, err := securitykey.ResolveSerial(cmd.Context(), newRunner(ios), store, ios, signingKeyFlags.SecurityKey)
		if err != nil {
			return err
		}
		if err := requireAllowedSerial(serial); err != nil {
			return err
		}
		refs, err := signingkey.Scan(dataDir, []string{serial}, signingKeyFlags.Username)
		if err != nil {
			return err
		}
		if len(refs) == 0 {
			return fmt.Errorf("%w for YubiKey %s (run `dotty signing-key new`)", signingkey.ErrKeyNotFound, serial)
		}

		if gitMode {
			preferDefaultKeyType(refs)
			_, pub, err := signingkey.Read(refs[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(ios.Out, signingkey.FormatGitKey(pub))
			return nil
		}

		for _, ref := range refs {
			priv, pub, err := signingkey.Read(ref)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(ios.Out, string(priv))
			_, _ = fmt.Fprint(ios.Out, string(pub))
		}
		return nil
	},
}

func init() {
	signingKeyGetCmd.Flags().StringVar(&signingKeyGetFlags.Format, "format", "text",
		"output format: text (stub + public key) or key (git literal key line)")
	signingKeyCmd.AddCommand(signingKeyGetCmd)
}
