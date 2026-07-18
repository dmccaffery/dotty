// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// SigningKeyImportFlags holds the flags for `dotty signing-key import`.
type SigningKeyImportFlags struct {
	Remove bool
}

var signingKeyImportFlags = SigningKeyImportFlags{}

var signingKeyImportCmd = &cobra.Command{
	Use:   "import <path> [--rm]",
	Short: "Import existing SSH signing-key stubs that match a connected security key.",
	Long: `Import key-handle stubs from elsewhere on disk into the private data
directory. Each stub is verified against the connected YubiKey(s): dotty
downloads the resident credentials from the hardware with ssh-keygen -K (which
prompts for the FIDO PIN and a touch) and keeps only the stubs whose public key
is actually resident on a connected key, filing each under that key's serial.
Stubs that match no connected key are skipped with a warning; if nothing
matches, import fails.

<path> may be a single stub file or a directory, which is walked recursively so
both flat and <serial>/id_*_sk_* layouts work. The secret never leaves the
hardware — only the key-handle stub and its public key are copied.

With several YubiKeys connected, dotty asks you to touch each in turn; touch the
key whose serial it names (etched on the key). --security-key narrows the import
to one connected key. Aliases are not set here — add them with
` + "`dotty security-key add`" + ` afterwards.`,
	Example: `  dotty signing-key import ./backup
  dotty signing-key import ./backup --rm
  dotty signing-key import --security-key=work ~/keys/id_ed25519_sk_deavon`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return importSigningKeys(cmd.Context(), cli.System(),
			args[0], signingKeyFlags.Username, signingKeyImportFlags.Remove)
	},
}

// importSigningKeys is the full `signing-key import` flow: verify stubs at
// path against the connected YubiKeys and file the matches. init's
// security-key step calls it too.
func importSigningKeys(ctx context.Context, ios cli.IOStreams, path, username string, remove bool) error {
	runner := newRunner(ios)

	if !ios.IsInteractive() {
		return fmt.Errorf("import needs an interactive terminal: ssh-keygen prompts for the FIDO PIN and a touch")
	}

	dataDir, err := cli.DataDir()
	if err != nil {
		return err
	}
	store, err := keyStore()
	if err != nil {
		return err
	}

	srcRefs, err := signingkey.ScanDir(path)
	if err != nil {
		return err
	}
	if username != "" {
		srcRefs = filterByUser(srcRefs, username)
	}
	if len(srcRefs) == 0 {
		return fmt.Errorf("no signing-key stubs found at %s", path)
	}

	serials, err := importTargetSerials(ctx, runner, store)
	if err != nil {
		return err
	}
	if serials, err = filterAllowedSerials(serials); err != nil {
		return err
	}
	if len(serials) == 0 {
		return errors.New("no connected YubiKey is allowed for the active profile (see dotty security-key allow)")
	}

	residentBySerial := make(map[string][]string, len(serials))
	for _, serial := range serials {
		scratch, err := os.MkdirTemp("", "dotty-import-")
		if err != nil {
			return fmt.Errorf("create scratch dir: %w", err)
		}
		defer func() { _ = os.RemoveAll(scratch) }()

		tui.Infof(ios, "Touch YubiKey %s when ssh-keygen asks, and enter its FIDO PIN", serial)
		ids, err := signingkey.ResidentPubKeys(ctx, runner, scratch)
		if err != nil {
			tui.Warnf(ios, "could not read resident keys from YubiKey %s: %v", serial, err)
			continue
		}
		residentBySerial[serial] = ids
	}

	imported, skipped, err := signingkey.Import(srcRefs, residentBySerial, dataDir, func(dest string) (bool, error) {
		ok, err := tui.Confirm(ios,
			fmt.Sprintf("A stub already exists at %s. Replace it?", dest),
			"The existing stub is overwritten with the imported one.")
		if errors.Is(err, tui.ErrAborted) {
			return false, nil
		}
		return ok, err
	})
	if err != nil {
		return err
	}

	for _, ref := range skipped {
		tui.Warnf(ios, "skipping %s: no connected YubiKey holds this credential", ref.PrivPath)
	}
	if len(imported) == 0 {
		return fmt.Errorf("no stubs at %s matched a connected YubiKey", path)
	}

	if err := removeImportedSources(ios, imported, remove); err != nil {
		return err
	}

	tui.Successf(ios, "Imported %d signing key(s)", len(imported))
	for _, imp := range imported {
		tui.Infof(ios, "%s → YubiKey %s (%s, %s)", imp.Dest, imp.Serial, imp.Source.Type, imp.Source.User)
	}
	return nil
}

// filterByUser keeps only the stubs enrolled under user.
func filterByUser(refs []signingkey.KeyRef, user string) []signingkey.KeyRef {
	var out []signingkey.KeyRef
	for _, ref := range refs {
		if ref.User == user {
			out = append(out, ref)
		}
	}
	return out
}

// importTargetSerials returns the connected serials to verify stubs against.
// --security-key narrows to one key, which must be connected; otherwise every
// connected key is a candidate so stubs for any of them import in one pass.
func importTargetSerials(ctx context.Context, runner *cli.ExecRunner, store *securitykey.Store) ([]string, error) {
	serials, err := securitykey.ListSerials(ctx, runner)
	if err != nil {
		return nil, err
	}
	if len(serials) == 0 {
		return nil, securitykey.ErrNoKeyPresent
	}

	ref := signingKeyFlags.SecurityKey
	if ref == "" {
		return serials, nil
	}
	want := ref
	if !securitykey.IsSerial(ref) {
		if want, err = store.ResolveName(ref); err != nil {
			return nil, err
		}
	}
	if !slices.Contains(serials, want) {
		return nil, fmt.Errorf("YubiKey %s is not connected", want)
	}
	return []string{want}, nil
}

// removeImportedSources deletes the originals of the imported stubs when --rm is
// set, or after an interactive confirmation otherwise. Declining (or esc) keeps
// them.
func removeImportedSources(ios cli.IOStreams, imported []signingkey.Imported, force bool) error {
	if !force {
		ok, err := tui.Confirm(ios,
			fmt.Sprintf("Remove %d imported source file(s)?", len(imported)),
			"Deletes the originals you imported from; the copies in the data dir are kept.")
		if errors.Is(err, tui.ErrAborted) || (err == nil && !ok) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	for _, imp := range imported {
		for _, p := range []string{imp.Source.PrivPath, imp.Source.PubPath} {
			if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("remove %s: %w", p, err)
			}
		}
	}
	return nil
}

func init() {
	signingKeyImportCmd.Flags().BoolVar(&signingKeyImportFlags.Remove, "rm", false,
		"remove the source stub files after a successful import")
	signingKeyCmd.AddCommand(signingKeyImportCmd)
}
