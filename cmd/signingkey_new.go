// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// SigningKeyNewFlags holds the flags for `dotty signing-key new`.
type SigningKeyNewFlags struct {
	Type string
}

var signingKeyNewFlags = SigningKeyNewFlags{}

var signingKeyNewCmd = &cobra.Command{
	Use:   "new [--type=<ed25519|ecdsa>]",
	Short: "Create a resident SSH signing key on a security key.",
	Long: `Enroll a new resident, PIN-protected (verify-required, no-touch-required)
SSH key on a YubiKey via ssh-keygen, filing the key-handle stub under the
key's serial in the private data directory.

With several YubiKeys plugged in, dotty asks you to unplug and re-insert the
intended key — the only reliable way to map a serial to a FIDO device, since
YubiKeys expose no USB serial. Esc falls back to a picker; ssh-keygen's own
touch-select then chooses the hardware, so touch the key whose serial dotty
names (it is etched on the key).

Re-enrolling an existing username replaces the resident credential on the
device as well as the stub.`,
	Example: `  dotty signing-key new
  dotty signing-key new --security-key=work --type=ecdsa
  dotty signing-key new --username=deavon`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return enrollSigningKey(cmd.Context(), cli.System(),
			signingKeyNewFlags.Type, signingKeyFlags.Username, signingKeyFlags.SecurityKey)
	},
}

// enrollSigningKey is the full `signing-key new` flow: enroll a resident key
// on the selected YubiKey, file the stub, and trust the result. init's
// security-key step calls it too. username and securityKeyRef may be empty
// (current OS user; interactive device selection).
func enrollSigningKey(ctx context.Context, ios cli.IOStreams, typ, username, securityKeyRef string) error {
	runner := newRunner(ios)
	dataDir, err := cli.DataDir()
	if err != nil {
		return err
	}
	store, err := keyStore()
	if err != nil {
		return err
	}

	if err := signingkey.ValidateType(typ); err != nil {
		return err
	}
	if username == "" {
		if username, err = resolveUsername(); err != nil {
			return err
		}
	}

	wantSerial := ""
	if ref := securityKeyRef; ref != "" {
		if securitykey.IsSerial(ref) {
			wantSerial = ref
		} else if wantSerial, err = store.ResolveName(ref); err != nil {
			return err
		}
	}
	device, err := securitykey.SelectDeviceForEnroll(ctx, runner, store, ios, wantSerial)
	if err != nil {
		return err
	}
	if err := requireAllowedSerial(device.Serial); err != nil {
		return err
	}

	stub := signingkey.StubPath(dataDir, device.Serial, typ, username)
	if err := cli.EnsureDir(signingkey.KeyDir(dataDir, device.Serial), 0o700); err != nil {
		return err
	}
	replaced, err := confirmReplaceStub(ios, stub, typ, username, device.Serial)
	if err != nil {
		return err
	}
	if !replaced {
		tui.Infof(ios, "Aborted; nothing changed")
		return nil
	}

	if device.Path == "" {
		tui.Infof(ios,
			"When ssh-keygen prompts for user presence, touch YubiKey %s (the serial is etched on the key)",
			device.Serial)
	}
	tui.Infof(ios, "ssh-keygen will ask for the key's FIDO PIN")
	if err := signingkey.Generate(ctx, runner, signingkey.KeygenOptions{
		Type:   typ,
		User:   username,
		Path:   stub,
		Device: device.Path,
	}); err != nil {
		return err
	}

	ref := signingkey.KeyRef{Serial: device.Serial, Type: typ, User: username, PrivPath: stub, PubPath: stub + ".pub"}
	_, pub, err := signingkey.Read(ref)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(ios.Out, string(pub))
	tui.Successf(ios, "Created %s signing key for %q on YubiKey %s", typ, username, device.Serial)
	tui.Infof(ios, "Stub: %s", stub)

	// Trust the new key (and any others on the plugged-in YubiKeys) so git can
	// verify what it signs. Best-effort: the key is already enrolled, so a
	// missing user.email or unwritable allowed_signers must not fail the run.
	refs, err := pluggedStubs(ctx, runner, dataDir, store, "", "")
	if err != nil {
		tui.Warnf(ios, "could not list keys to trust: %v", err)
	} else if err := trustKeys(ctx, ios, runner, "", refs); err != nil {
		tui.Warnf(ios, "could not update allowed_signers: %v", err)
	}

	tui.Infof(ios, "Set up git commit signing with: dotty signing-key sign --print-git-config")
	return nil
}

// confirmReplaceStub clears the way for a new stub: a missing one is a yes,
// an existing one asks before both files are removed (re-enrolling replaces
// the resident credential on the device, not just the stub on disk). false
// means the user backed out.
func confirmReplaceStub(ios cli.IOStreams, stub, typ, username, serial string) (bool, error) {
	if _, err := os.Stat(stub); errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	ok, err := tui.Confirm(ios,
		fmt.Sprintf("A %s key for %q already exists on YubiKey %s. Replace it?", typ, username, serial),
		"Re-enrolling replaces the resident credential on the device, not just the stub on disk.")
	if errors.Is(err, tui.ErrNotInteractive) {
		return false, fmt.Errorf("key stub %s already exists; remove it or run interactively to confirm replacing it", stub)
	}
	if err != nil && !errors.Is(err, tui.ErrAborted) {
		return false, err
	}
	if !ok {
		return false, nil
	}
	for _, p := range []string{stub, stub + ".pub"} {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("remove old stub: %w", err)
		}
	}
	return true, nil
}

// resolveUsername returns --username or the current OS user.
func resolveUsername() (string, error) {
	if signingKeyFlags.Username != "" {
		return signingKeyFlags.Username, signingkey.ValidateUser(signingKeyFlags.Username)
	}
	current, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("resolve current user (pass --username): %w", err)
	}
	return current.Username, signingkey.ValidateUser(current.Username)
}

func init() {
	signingKeyNewCmd.Flags().StringVar(&signingKeyNewFlags.Type, "type", "ed25519", "key type: ed25519 or ecdsa")
	signingKeyCmd.AddCommand(signingKeyNewCmd)
}
