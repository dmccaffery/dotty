// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

var signingKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the signing keys on plugged-in security keys.",
	Long: `Show the signing keys of all currently plugged-in YubiKeys in a
fuzzy-filterable table (serial, aliases, key type, username). Selecting a row
prints its private key stub and public key; esc exits without printing.
Unlike the other signing-key verbs, list never asks you to pick a key first.
Without a terminal the table prints plainly and nothing is selectable.`,
	Example: `  dotty signing-key list
  dotty signing-key list --username=deavon
  dotty ssh-key list --security-key=work`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		dataDir, err := cli.DataDir()
		if err != nil {
			return err
		}
		store, err := keyStore()
		if err != nil {
			return err
		}

		plugged, err := securitykey.ListSerials(cmd.Context(), newRunner(ios))
		if err != nil {
			return err
		}
		serials := plugged
		if ref := signingKeyFlags.SecurityKey; ref != "" {
			want := ref
			if !securitykey.IsSerial(ref) {
				if want, err = store.ResolveName(ref); err != nil {
					return err
				}
			}
			serials = nil
			if slices.Contains(plugged, want) {
				serials = []string{want}
			}
		}
		if len(serials) == 0 {
			tui.Infof(ios, "No matching YubiKeys plugged in")
			return nil
		}

		refs, err := signingkey.Scan(dataDir, serials, signingKeyFlags.Username)
		if err != nil {
			return err
		}
		if len(refs) == 0 {
			tui.Infof(ios, "No signing keys found for the plugged-in YubiKeys (run `dotty signing-key new`)")
			return nil
		}

		aliases := store.AliasesBySerial()
		headers := []string{"SERIAL", "ALIASES", "TYPE", "USERNAME"}
		rows := make([]tui.TableRow, len(refs))
		for i, ref := range refs {
			var names []string
			for _, a := range aliases[ref.Serial] {
				names = append(names, a.Name)
			}
			rows[i] = tui.TableRow{
				Cells: []string{ref.Serial, strings.Join(names, ", "), ref.Type, ref.User},
				Value: ref.PrivPath,
			}
		}

		if !ios.IsInteractive() {
			_, _ = fmt.Fprint(ios.Out, tui.RenderTable(headers, rows))
			return nil
		}

		value, ok, err := tui.FilterTable(ios, "Signing keys (enter prints the key, esc exits)", headers, rows)
		if errors.Is(err, tui.ErrNotInteractive) || !ok {
			return nil
		}
		if err != nil {
			return err
		}
		for _, ref := range refs {
			if ref.PrivPath == value {
				priv, pub, err := signingkey.Read(ref)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprint(ios.Out, string(priv))
				_, _ = fmt.Fprint(ios.Out, string(pub))
				return nil
			}
		}
		return nil
	},
}

func init() {
	signingKeyCmd.AddCommand(signingKeyListCmd)
}
