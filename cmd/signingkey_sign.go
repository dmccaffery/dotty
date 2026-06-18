// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
)

// signOwnFlags is the ExtractFlags spec for the sign proxy: the flags dotty
// owns; everything else forwards to ssh-keygen verbatim.
var signOwnFlags = map[string]bool{
	"security-key":     true,
	"username":         true,
	"print-git-config": false,
}

var signingKeySignCmd = &cobra.Command{
	Use:   "sign [ssh-keygen args] [file ...]",
	Short: "Sign a payload with a signing key (ssh-keygen proxy).",
	Long: `Proxy to ssh-keygen -Y sign using a hardware-backed signing key. Built for
git: point gpg.ssh.program at dotty (or a dotty-ssh-sign symlink) and git's
own arguments pass straight through; when git supplies a literal public key
via gpg.ssh.defaultKeyCommand, dotty swaps in the matching stub so the
YubiKey signs. Run with --print-git-config for ready-to-paste setup.

Humans can sign files too: with no -f, dotty resolves the key from
--security-key/--username (or the single plugged-in YubiKey) and defaults
the namespace to "file".`,
	Example: `  dotty signing-key sign --print-git-config
  dotty signing-key sign document.txt
  dotty signing-key sign --security-key=work -n release artifact.tar.gz`,
	// Verbatim passthrough: pflag would mangle ssh-keygen's short flags, so
	// parsing is off and --help is handled manually via ExtractFlags.
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		own, rest, help := cli.ExtractFlags(args, signOwnFlags)
		if help {
			return cmd.Help()
		}
		ios := cli.System()
		if own["print-git-config"] == "true" {
			return printGitConfig(ios)
		}
		dataDir, err := cli.DataDir()
		if err != nil {
			return err
		}
		store, err := securitykey.LoadStore(securitykey.StorePath(dataDir))
		if err != nil {
			return err
		}

		// Resolved lazily — git always passes -f, so its calls never prompt.
		resolveDefault := func() (string, error) {
			serial, err := securitykey.ResolveSerial(cmd.Context(), newRunner(ios), store, ios, own["security-key"])
			if err != nil {
				return "", err
			}
			refs, err := signingkey.Scan(dataDir, []string{serial}, own["username"])
			if err != nil {
				return "", err
			}
			if len(refs) == 0 {
				return "", fmt.Errorf("%w for YubiKey %s (run `dotty signing-key new`)", signingkey.ErrKeyNotFound, serial)
			}
			if len(refs) > 1 {
				return "", fmt.Errorf("YubiKey %s has %d signing keys; disambiguate with --username", serial, len(refs))
			}
			return refs[0].PrivPath, nil
		}
		scan := func() ([]signingkey.KeyRef, error) {
			return signingkey.Scan(dataDir, nil, "")
		}

		finalArgs, err := signingkey.RewriteSignArgs(rest, resolveDefault, scan, os.ReadFile)
		if err != nil {
			return err
		}
		return signingkey.Sign(cmd.Context(), newRunner(ios), finalArgs)
	},
}

// printGitConfig prints ready-to-paste git configuration for signing commits
// through dotty.
func printGitConfig(ios cli.IOStreams) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate the dotty binary: %w", err)
	}
	username, err := resolveUsername()
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, `# Sign git commits and tags with your YubiKey through dotty:
git config --global gpg.format ssh
git config --global gpg.ssh.program %q
git config --global gpg.ssh.defaultKeyCommand "%s signing-key get --format=key --username %s"
git config --global commit.gpgsign true

# Verification needs an allowed-signers file, e.g.:
#   echo "$(git config user.email) $(dotty signing-key get --format=key --username %s | sed s/^key:://)" \
#     >> ~/.ssh/allowed_signers
#   git config --global gpg.ssh.allowedSignersFile ~/.ssh/allowed_signers
`, exe, exe, username, username)
	return nil
}

func init() {
	signingKeyCmd.AddCommand(signingKeySignCmd)
}
