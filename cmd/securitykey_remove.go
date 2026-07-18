// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// SecurityKeyRemoveFlags holds the flags for `dotty security-key remove`.
type SecurityKeyRemoveFlags struct {
	Name string
}

var securityKeyRemoveFlags = SecurityKeyRemoveFlags{}

var securityKeyRemoveCmd = &cobra.Command{
	Use:     "remove [--name=<name>]",
	Aliases: []string{"rm"},
	Short:   "Remove security key aliases.",
	Long: `Remove one alias by --name, or pick aliases interactively: a tree grouped
by serial number — collapsible with h/l, filterable with / — where space
selects and enter confirms. A key may carry several aliases, so multiple
selections are removed in one go.`,
	Example: `  dotty security-key remove --name=old-key
  dotty security-key remove
  dotty sk --serial=12345678 rm`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		store, err := keyStore()
		if err != nil {
			return err
		}

		if name := securityKeyRemoveFlags.Name; name != "" {
			serial, err := store.ResolveName(name)
			if err != nil {
				return err
			}
			if want := securityKeyFlags.Serial; want != "" && want != serial {
				return fmt.Errorf("alias %q names serial %s, not %s", name, serial, want)
			}
			store.Remove(name)
			if err := store.Save(); err != nil {
				return err
			}
			tui.Successf(ios, "Removed alias %q (YubiKey %s)", name, serial)
			return nil
		}

		bySerial := store.AliasesBySerial()
		if want := securityKeyFlags.Serial; want != "" {
			for serial := range bySerial {
				if serial != want {
					delete(bySerial, serial)
				}
			}
		}
		if len(bySerial) == 0 {
			tui.Infof(ios, "No aliases to remove")
			return nil
		}

		nodes := aliasTree(cmd, ios, bySerial)
		names, err := tui.TreeMultiSelect(ios, "Remove which aliases?", nodes)
		if errors.Is(err, tui.ErrAborted) {
			return nil
		}
		if errors.Is(err, tui.ErrNotInteractive) {
			return errors.New("no terminal for the picker; pass --name")
		}
		if err != nil {
			return err
		}
		if len(names) == 0 {
			tui.Infof(ios, "Nothing selected; nothing removed")
			return nil
		}

		ok, err := tui.Confirm(ios, fmt.Sprintf("Remove %d alias%s?", len(names), plural(len(names), "", "es")), "")
		if err != nil && !errors.Is(err, tui.ErrAborted) {
			return err
		}
		if !ok {
			tui.Infof(ios, "Aborted; nothing removed")
			return nil
		}
		store.Remove(names...)
		if err := store.Save(); err != nil {
			return err
		}
		tui.Successf(ios, "Removed %d alias%s", len(names), plural(len(names), "", "es"))
		return nil
	},
}

// aliasTree builds the serial-grouped tree, badging serials that are
// currently plugged in (best effort — enumeration failures just drop the
// badge).
func aliasTree(cmd *cobra.Command, ios cli.IOStreams, bySerial map[string][]securitykey.Alias) []tui.TreeNode {
	plugged := map[string]bool{}
	if serials, err := securitykey.ListSerials(cmd.Context(), newRunner(ios)); err == nil {
		for _, s := range serials {
			plugged[s] = true
		}
	}

	serials := slices.Sorted(maps.Keys(bySerial))

	nodes := make([]tui.TreeNode, 0, len(serials))
	for _, serial := range serials {
		label := serial
		if plugged[serial] {
			label += "  (plugged in)"
		}
		node := tui.TreeNode{Label: label}
		for _, alias := range bySerial[serial] {
			leafLabel := alias.Name
			if alias.Description != "" {
				leafLabel = fmt.Sprintf("%s — %s", alias.Name, alias.Description)
			}
			node.Leaves = append(node.Leaves, tui.TreeLeaf{Label: leafLabel, Value: alias.Name})
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func init() {
	securityKeyRemoveCmd.Flags().StringVar(&securityKeyRemoveFlags.Name, "name", "", "alias to remove")
	securityKeyCmd.AddCommand(securityKeyRemoveCmd)
}
