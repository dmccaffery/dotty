// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
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
		ios := cli.System()
		runner := newRunner(ios)
		dataDir, err := cli.DataDir()
		if err != nil {
			return err
		}
		store, err := securitykey.LoadStore(securitykey.StorePath(dataDir))
		if err != nil {
			return err
		}

		typ := signingKeyNewFlags.Type
		if err := signingkey.ValidateType(typ); err != nil {
			return err
		}
		username, err := resolveUsername()
		if err != nil {
			return err
		}

		wantSerial := ""
		if ref := signingKeyFlags.SecurityKey; ref != "" {
			if securitykey.IsSerial(ref) {
				wantSerial = ref
			} else if wantSerial, err = store.ResolveName(ref); err != nil {
				return err
			}
		}
		device, err := securitykey.SelectDeviceForEnroll(cmd.Context(), runner, store, ios, wantSerial)
		if err != nil {
			return err
		}

		stub := signingkey.StubPath(dataDir, device.Serial, typ, username)
		if err := cli.EnsureDir(signingkey.KeyDir(dataDir, device.Serial), 0o700); err != nil {
			return err
		}
		if _, err := os.Stat(stub); !errors.Is(err, fs.ErrNotExist) {
			ok, err := tui.Confirm(ios,
				fmt.Sprintf("A %s key for %q already exists on YubiKey %s. Replace it?", typ, username, device.Serial),
				"Re-enrolling replaces the resident credential on the device, not just the stub on disk.")
			if errors.Is(err, tui.ErrNotInteractive) {
				return fmt.Errorf("key stub %s already exists; remove it or run interactively to confirm replacing it", stub)
			}
			if err != nil && !errors.Is(err, tui.ErrAborted) {
				return err
			}
			if !ok {
				tui.Infof(ios, "Aborted; nothing changed")
				return nil
			}
			for _, p := range []string{stub, stub + ".pub"} {
				if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf("remove old stub: %w", err)
				}
			}
		}

		if device.Path == "" {
			tui.Infof(ios,
				"When ssh-keygen prompts for user presence, touch YubiKey %s (the serial is etched on the key)",
				device.Serial)
		}
		tui.Infof(ios, "ssh-keygen will ask for the key's FIDO PIN")
		if err := signingkey.Generate(cmd.Context(), runner, signingkey.KeygenOptions{
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
		tui.Infof(ios, "Set up git commit signing with: dotty signing-key sign --print-git-config")
		return nil
	},
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
