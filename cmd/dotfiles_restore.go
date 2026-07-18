// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/linker"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// DotfilesRestoreFlags holds the flags for `dotty dotfiles restore`.
type DotfilesRestoreFlags struct {
	Timestamp string
}

var dotfilesRestoreFlags = DotfilesRestoreFlags{}

var dotfilesRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Copy a backup set back over the links that displaced it.",
	Long: `Restore the files a link run backed up: every file in the chosen backup set
is copied back to the absolute path it came from, replacing the symlink that
displaced it. Without --timestamp, an interactive picklist offers the
available sets, newest first. The set is copied, not consumed — a restore can
be repeated.`,
	Example: `  dotty dotfiles restore
  dotty dotfiles restore --timestamp=2026-07-15T10-30-00`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		dataDir, err := cli.DataDir()
		if err != nil {
			return err
		}
		root := filepath.Join(dataDir, "backups")

		ts := dotfilesRestoreFlags.Timestamp
		if ts == "" {
			ts, err = pickBackupSet(ios, root)
			if errors.Is(err, tui.ErrAborted) || ts == "" && err == nil {
				return nil
			}
			if err != nil {
				return err
			}
		}

		restored, err := linker.Restore(filepath.Join(root, ts))
		if err != nil {
			return err
		}
		tui.Successf(ios, "Restored %d files from %s", len(restored), filepath.Join(root, ts))
		return nil
	},
}

func init() {
	dotfilesRestoreCmd.Flags().StringVar(&dotfilesRestoreFlags.Timestamp, "timestamp", "",
		"backup set to restore (a directory name under the backups dir)")
	dotfilesCmd.AddCommand(dotfilesRestoreCmd)
}

// pickBackupSet lists the backup sets newest-first and asks which to restore.
func pickBackupSet(ios cli.IOStreams, root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("read backups dir %s: %w", root, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no backup sets under %s", root)
	}
	options := make([]tui.Option, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- { // lexicographic timestamps: newest last
		if entries[i].IsDir() {
			options = append(options, tui.Option{Label: entries[i].Name(), Value: entries[i].Name()})
		}
	}
	if len(options) == 0 {
		return "", fmt.Errorf("no backup sets under %s", root)
	}
	picked, err := tui.FuzzySelect(ios, "Restore which backup set?", options)
	if errors.Is(err, tui.ErrNotInteractive) {
		return "", errors.New("several backup sets exist; pass --timestamp")
	}
	return picked, err
}
