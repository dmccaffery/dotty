// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bitwise-media-group/dotty/internal/git"
)

// noGitConfigAnnotation marks a flag that must never pick up a git-config
// default — reserved for toggles too destructive to enable persistently,
// like resign --root.
const noGitConfigAnnotation = "dotty-no-git-config"

// excludeGitConfigFlags opts the named flags out of git-config defaults.
// An unknown flag name is a wiring bug, so it fails loudly at init time.
func excludeGitConfigFlags(fs *pflag.FlagSet, names ...string) {
	for _, name := range names {
		if err := fs.SetAnnotation(name, noGitConfigAnnotation, []string{"true"}); err != nil {
			panic(fmt.Sprintf("exclude flag from git config: %v", err))
		}
	}
}

// applyGitConfigFlagDefaults fills cmd's own flags from git configuration:
// flag --name on verb <verb> reads dotty.<verb>.<name>. Config supplies
// defaults, not arguments — a flag given on the command line keeps its value,
// and a config-sourced value does not mark the flag as changed, so
// flag-interaction checks (Flags().Changed) still describe only the command
// line. Hidden flags, the help flag, positional arguments, and flags opted
// out via excludeGitConfigFlags never read config. Boolean flags accept git's
// boolean vocabulary; an unparseable value is an error naming the key.
func applyGitConfigFlagDefaults(ctx context.Context, r git.Runner, cmd *cobra.Command) error {
	var firstErr error
	cmd.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if firstErr != nil || f.Changed || f.Hidden || f.Name == "help" {
			return
		}
		if _, excluded := f.Annotations[noGitConfigAnnotation]; excluded {
			return
		}
		key := "dotty." + cmd.Name() + "." + f.Name
		var value string
		var found bool
		var err error
		if f.Value.Type() == "bool" {
			value, found, err = git.ConfigLookupBool(ctx, r, key)
		} else {
			value, found, err = git.ConfigLookup(ctx, r, key)
		}
		if err != nil {
			firstErr = err
			return
		}
		if !found {
			return
		}
		if err := f.Value.Set(value); err != nil {
			firstErr = fmt.Errorf("git config %s = %q: %w", key, value, err)
		}
	})
	return firstErr
}
