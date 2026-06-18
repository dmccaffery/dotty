// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/env"
)

// envRunOwnFlags is the ExtractFlags spec for the run proxy: the flags dotty
// owns. Everything else — the command and its own flags — forwards verbatim.
var envRunOwnFlags = map[string]bool{"namespace": true, "in-file": true}

var envRunCmd = &cobra.Command{
	Use:   "run [--in-file=<file>] -- <command> [args...]",
	Short: "Run a command with a namespace's credentials in its environment.",
	Long: `Launch a command with every credential in the namespace exported as an
environment variable, the way op run does. dotty parses its own --namespace and
--in-file (and --help); everything after -- is the command and its arguments,
passed through untouched. Put dotty's flags before -- (use -- when the command
takes a --namespace of its own). The command inherits the terminal, and dotty
exits with its exit code.

With --in-file, the environment is built from a .env template instead of the
whole namespace: every {{ dotty://<namespace>/KEY }} reference is resolved from
the keychain and every plain KEY=value assignment is passed through, the way env
use fills a template — but the secrets are handed straight to the process and
never written to disk.`,
	Example: `  dotty env run --namespace=aws -- aws s3 ls
  dotty env run --namespace=ci -- ./deploy.sh
  dotty env run --in-file=.env.tmpl -- ./serve`,
	// Verbatim passthrough: the child owns its flags, so parsing is off and
	// --help is handled manually via ExtractFlags.
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		own, rest, help := cli.ExtractFlags(args, envRunOwnFlags)
		if help {
			return cmd.Help()
		}
		ios := cli.System()

		namespace := envFlags.Namespace
		if ns, ok := own["namespace"]; ok {
			namespace = ns
		}

		if len(rest) > 0 && rest[0] == "--" {
			rest = rest[1:]
		}
		if len(rest) == 0 {
			return errors.New("no command given; usage: dotty env run --namespace=<ns> -- <command> [args...]")
		}

		extraEnv, err := envRunEnviron(cmd.Context(), ios, namespace, own["in-file"])
		if err != nil {
			return err
		}

		return newRunner(ios).RunInteractiveEnv(cmd.Context(), extraEnv, rest[0], rest[1:]...)
	},
}

// envRunEnviron builds the KEY=value pairs the child process inherits. With an
// inFile the environment comes from that .env template: every reference is
// resolved from the keychain and every plain assignment is passed through, so
// the secrets reach the process without ever touching disk, and file order is
// preserved so a later duplicate key wins per os/exec semantics. Without an
// inFile it falls back to every credential stored in the namespace, sorted for
// a deterministic order.
func envRunEnviron(ctx context.Context, ios cli.IOStreams, namespace, inFile string) ([]string, error) {
	store := newEnvStore(ios)

	if inFile != "" {
		src, err := os.ReadFile(inFile)
		if err != nil {
			return nil, fmt.Errorf("read env file: %w", err)
		}
		entries, err := env.Parse(string(src), store.Resolver(ctx, namespace))
		if err != nil {
			return nil, err
		}
		extraEnv := make([]string, 0, len(entries))
		for _, e := range entries {
			extraEnv = append(extraEnv, e.Key+"="+e.Value)
		}
		return extraEnv, nil
	}

	secrets, err := store.All(ctx, namespace)
	if err != nil {
		return nil, err
	}
	extraEnv := make([]string, 0, len(secrets))
	for k, v := range secrets {
		extraEnv = append(extraEnv, k+"="+v)
	}
	sort.Strings(extraEnv) // deterministic order; later duplicates would win regardless
	return extraEnv, nil
}

func init() {
	envCmd.AddCommand(envRunCmd)
}
