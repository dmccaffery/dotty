// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// SecurityKeyAddFlags holds the flags for `dotty security-key add`.
type SecurityKeyAddFlags struct {
	Name        string
	Description string
}

var securityKeyAddFlags = SecurityKeyAddFlags{}

var securityKeyAddCmd = &cobra.Command{
	Use:   "add [--name=<name>] [--description=<description>]",
	Short: "Add a named alias for a security key.",
	Long: `Register a memorable alias for a security key's serial number. Without
--serial, the plugged-in key is used (or picked from a list when several are
present, with the option to type a serial by hand). Alias names are unique
across all keys. Without --description, dotty offers to open $EDITOR for one;
that step can be skipped.`,
	Example: `  dotty security-key add
  dotty security-key --serial=12345678 add --name=work
  dotty sk add --name=backup --description="kept in the safe"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		store, err := keyStore()
		if err != nil {
			return err
		}

		serial := securityKeyFlags.Serial
		if serial == "" {
			serial, err = pickSerialForAdd(cmd, ios, store)
			if err != nil {
				return err
			}
		}
		if !securitykey.IsSerial(serial) {
			return fmt.Errorf("--serial %q must be a numeric serial number", serial)
		}

		name := securityKeyAddFlags.Name
		if name == "" {
			name, err = tui.Input(ios, fmt.Sprintf("Alias name for YubiKey %s", serial), "work", func(s string) error {
				if err := securitykey.ValidateName(s); err != nil {
					return err
				}
				if _, err := store.ResolveName(s); err == nil {
					return fmt.Errorf("alias %q is already in use", s)
				}
				return nil
			})
			if errors.Is(err, tui.ErrNotInteractive) {
				return errors.New("no alias name given; pass --name or run interactively")
			}
			if err != nil {
				return err
			}
		}

		description := securityKeyAddFlags.Description
		if description == "" && ios.IsInteractive() {
			ok, err := tui.Confirm(ios, "Open the editor to add a description?", "Skip with N — the alias works without one.")
			if err != nil && !errors.Is(err, tui.ErrAborted) {
				return err
			}
			if ok {
				description, err = cli.EditTempFile(cmd.Context(), newRunner(ios), "")
				if err != nil {
					return err
				}
			}
		}

		if err := store.Add(serial, name, description); err != nil {
			return err
		}
		if err := store.Save(); err != nil {
			return err
		}
		tui.Successf(ios, "Added alias %q for YubiKey %s", name, serial)
		return nil
	},
}

// pickSerialForAdd resolves the serial interactively: the single plugged-in
// key, a picker over several, or manual entry — aliases may be registered for
// keys that are not plugged in.
func pickSerialForAdd(cmd *cobra.Command, ios cli.IOStreams, store *securitykey.Store) (string, error) {
	const manual = ""
	serials, err := securitykey.ListSerials(cmd.Context(), newRunner(ios))
	if err != nil {
		tui.Warnf(ios, "Could not enumerate YubiKeys (%v)", err)
	}

	switch {
	case len(serials) == 1:
		tui.Infof(ios, "Using the plugged-in YubiKey %s", serials[0])
		return serials[0], nil
	case len(serials) > 1 && ios.IsInteractive():
		options := make([]tui.Option, 0, len(serials)+1)
		for _, s := range serials {
			options = append(options, tui.Option{Label: securitykey.SerialLabel(store, s), Value: s})
		}
		options = append(options, tui.Option{Label: "(enter a serial manually)", Value: manual})
		choice, err := tui.FuzzySelect(ios, "Alias which YubiKey?", options)
		if err != nil {
			return "", err
		}
		if choice != manual {
			return choice, nil
		}
	case !ios.IsInteractive():
		if len(serials) == 0 {
			return "", securitykey.ErrNoKeyPresent
		}
		return "", securitykey.ErrAmbiguousKey
	}

	serial, err := tui.Input(ios, "Security key serial number", "12345678", func(s string) error {
		if !securitykey.IsSerial(s) {
			return errors.New("serial numbers are numeric")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return serial, nil
}

func init() {
	securityKeyAddCmd.Flags().StringVar(&securityKeyAddFlags.Name, "name", "", "alias name (unique across all keys)")
	securityKeyAddCmd.Flags().StringVar(&securityKeyAddFlags.Description, "description", "", "what this key is for")
	securityKeyCmd.AddCommand(securityKeyAddCmd)
}
