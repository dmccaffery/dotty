// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/git"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// defaultAllowedSigners is the allowed_signers file dotty writes when git's
// gpg.ssh.allowedSignersFile is unset.
const defaultAllowedSigners = "~/.ssh/allowed_signers"

// SigningKeyTrustFlags holds the flags for `dotty signing-key trust`.
type SigningKeyTrustFlags struct {
	Path string
}

var signingKeyTrustFlags = SigningKeyTrustFlags{}

var signingKeyTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Trust plugged-in keys' signatures in the git allowed_signers file.",
	Long: `Append every signing key on the plugged-in YubiKeys to your OpenSSH
allowed_signers file, so git can verify the commits and tags they sign. Each
entry pairs your committer email (git config user.email) with the key:

  you@example.com sk-ssh-ed25519@openssh.com AAAA...

The file comes from git's gpg.ssh.allowedSignersFile, falling back to
~/.ssh/allowed_signers (dotty then prints the one-liner to point git at it).
~/.ssh (0700) and the file (0600) are created if missing; existing entries are
kept and a key already trusted for your email is left alone, so re-running is
safe. --security-key and --username narrow which stubs are added; --path writes
a different file.`,
	Example: `  dotty signing-key trust
  dotty signing-key trust --username=deavon
  dotty signing-key trust --path=~/.config/git/allowed_signers`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		runner := newRunner(ios)
		dataDir, err := cli.DataDir()
		if err != nil {
			return err
		}
		store, err := keyStore()
		if err != nil {
			return err
		}
		refs, err := pluggedStubs(cmd.Context(), runner, dataDir, store,
			signingKeyFlags.SecurityKey, signingKeyFlags.Username)
		if err != nil {
			return err
		}
		return trustKeys(cmd.Context(), ios, runner, signingKeyTrustFlags.Path, refs)
	},
}

// pluggedStubs returns the signing-key stubs on the currently plugged-in
// YubiKeys. securityKeyRef ("" for all) narrows to one connected key by serial
// or alias, and must name a connected key; user ("" for all) narrows to one
// username.
func pluggedStubs(
	ctx context.Context, runner *cli.ExecRunner, dataDir string, store *securitykey.Store,
	securityKeyRef, user string,
) ([]signingkey.KeyRef, error) {
	plugged, err := securitykey.ListSerials(ctx, runner)
	if err != nil {
		return nil, err
	}
	if len(plugged) == 0 {
		return nil, securitykey.ErrNoKeyPresent
	}
	serials := plugged
	if securityKeyRef != "" {
		want := securityKeyRef
		if !securitykey.IsSerial(securityKeyRef) {
			if want, err = store.ResolveName(securityKeyRef); err != nil {
				return nil, err
			}
		}
		if !slices.Contains(plugged, want) {
			return nil, fmt.Errorf("YubiKey %s is not connected", want)
		}
		serials = []string{want}
	}
	if serials, err = filterAllowedSerials(serials); err != nil {
		return nil, err
	}
	if len(serials) == 0 {
		return nil, errors.New("no connected YubiKey is allowed for the active profile (see dotty security-key allow)")
	}
	return signingkey.Scan(dataDir, serials, user)
}

// trustKeys writes refs into the allowed_signers file for the git committer
// email and reports the outcome. It is the shared core of the trust command and
// the automatic sync at the end of `new`. override ("" to resolve from git
// config) forces a specific allowed_signers path.
func trustKeys(
	ctx context.Context, ios cli.IOStreams, runner *cli.ExecRunner, override string, refs []signingkey.KeyRef,
) error {
	if len(refs) == 0 {
		return fmt.Errorf("no signing keys on the plugged-in YubiKey(s) (run `dotty signing-key new`)")
	}
	email, ok, err := git.ConfigLookup(ctx, runner, "user.email")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("git config user.email is not set: set it so allowed_signers can name the committer")
	}
	path, configured, err := resolveAllowedSigners(ctx, runner, override)
	if err != nil {
		return err
	}
	result, err := signingkey.Trust(path, email, refs)
	if err != nil {
		return err
	}
	for _, ref := range result.Added {
		tui.Successf(ios, "Trusted %s key for %s (YubiKey %s) in %s", ref.Type, ref.User, ref.Serial, path)
	}
	for _, ref := range result.Skipped {
		tui.Infof(ios, "%s key for %s (YubiKey %s) already trusted", ref.Type, ref.User, ref.Serial)
	}
	// Writing the keys is pointless until git knows where to read them, so nudge
	// the user when the fallback path was used rather than a configured one.
	if !configured {
		tui.Infof(ios, "Point git at it: git config --global gpg.ssh.allowedSignersFile %s", path)
	}
	return nil
}

// resolveAllowedSigners returns the allowed_signers path and whether git's
// gpg.ssh.allowedSignersFile chose it. override (from --path) wins; otherwise
// the git config value; otherwise the ~/.ssh/allowed_signers fallback, for
// which configured is false so callers can prompt the user to set the config.
func resolveAllowedSigners(ctx context.Context, runner *cli.ExecRunner, override string) (string, bool, error) {
	if override != "" {
		path, err := cli.ExpandHome(override)
		return path, true, err
	}
	if configured, ok, err := git.ConfigLookup(ctx, runner, "gpg.ssh.allowedSignersFile"); err != nil {
		return "", false, err
	} else if ok {
		path, err := cli.ExpandHome(configured)
		return path, true, err
	}
	path, err := cli.ExpandHome(defaultAllowedSigners)
	return path, false, err
}

func init() {
	signingKeyTrustCmd.Flags().StringVar(&signingKeyTrustFlags.Path, "path", "",
		"allowed_signers file to write (default: git gpg.ssh.allowedSignersFile, else ~/.ssh/allowed_signers)")
	signingKeyCmd.AddCommand(signingKeyTrustCmd)
}
