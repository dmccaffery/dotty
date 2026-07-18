// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/profile"
	"github.com/bitwise-media-group/dotty/internal/scaffold"
	"github.com/bitwise-media-group/dotty/internal/securitykey"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

var securityKeyAllowCmd = &cobra.Command{
	Use:   "allow [<serial>|<alias>...]",
	Short: "Restrict a profile to specific security keys.",
	Long: `Add security keys to a profile's allowlist. Once a profile has one, its
machines refuse every other key for signing, linking, enrollment, and import
— so personal keys are never used against work devices, and vice versa.

The list applies to the active profile unless --profile names another. It is
a property of the machine class: it lives in the profile's profile.json,
travels with the dotfiles repository, and activating another profile swaps
it. Arguments are serials or aliases; without arguments, an interactive
picklist offers the known and connected keys. Removing every entry later
(dotty security-key disallow) lifts the restriction.`,
	Example: `  dotty security-key allow            # pick from known + connected keys
  dotty security-key allow work-key
  dotty --profile=work security-key allow 12345678`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		profileDir, answers, store, err := allowlistProfile()
		if err != nil {
			return err
		}

		serials, err := resolveSerialArgs(store, args)
		if err != nil {
			return err
		}
		if len(serials) == 0 {
			serials, err = pickAllowSerials(cmd.Context(), ios, store, answers)
			if err != nil || len(serials) == 0 {
				return err
			}
		}

		for _, serial := range serials {
			if !slices.Contains(answers.AllowedSerials, serial) {
				answers.AllowedSerials = append(answers.AllowedSerials, serial)
			}
		}
		slices.Sort(answers.AllowedSerials)
		if err := scaffold.SaveAnswers(profileDir, answers); err != nil {
			return err
		}
		tui.Successf(ios, "Profile %s now allows only: %s",
			answers.ProfileName, strings.Join(answers.AllowedSerials, ", "))
		return nil
	},
}

var securityKeyDisallowCmd = &cobra.Command{
	Use:   "disallow [<serial>|<alias>...]",
	Short: "Remove security keys from a profile's allowlist.",
	Long: `Remove security keys from a profile's allowlist (the active profile unless
--profile names another). Removing the last entry lifts the restriction —
the profile allows every key again.`,
	Example: `  dotty security-key disallow
  dotty --profile=work security-key disallow 12345678`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ios := cli.System()
		profileDir, answers, store, err := allowlistProfile()
		if err != nil {
			return err
		}
		if len(answers.AllowedSerials) == 0 {
			tui.Infof(ios, "Profile %s has no allowlist; every key is allowed", answers.ProfileName)
			return nil
		}

		serials, err := resolveSerialArgs(store, args)
		if err != nil {
			return err
		}
		if len(serials) == 0 {
			options := make([]tui.Option, len(answers.AllowedSerials))
			for i, serial := range answers.AllowedSerials {
				options[i] = tui.Option{Label: securitykey.SerialLabel(store, serial), Value: serial}
			}
			prompt := fmt.Sprintf("Disallow which keys for profile %s?", answers.ProfileName)
			serials, err = tui.MultiSelect(ios, prompt, options)
			if errors.Is(err, tui.ErrNotInteractive) {
				return errors.New("pass the serials or aliases to disallow")
			}
			if err != nil || len(serials) == 0 {
				return err
			}
		}

		answers.AllowedSerials = slices.DeleteFunc(answers.AllowedSerials, func(s string) bool {
			return slices.Contains(serials, s)
		})
		if err := scaffold.SaveAnswers(profileDir, answers); err != nil {
			return err
		}
		if len(answers.AllowedSerials) > 0 {
			tui.Successf(ios, "Profile %s now allows only: %s",
				answers.ProfileName, strings.Join(answers.AllowedSerials, ", "))
		} else {
			tui.Successf(ios, "Profile %s no longer restricts security keys", answers.ProfileName)
		}
		return nil
	},
}

func init() {
	securityKeyCmd.AddCommand(securityKeyAllowCmd)
	securityKeyCmd.AddCommand(securityKeyDisallowCmd)
}

// allowlistProfile resolves the profile the allowlist verbs act on
// (--profile, else the active profile) and loads its answers — a profile
// that has none yet starts empty — plus the alias store for name resolution.
func allowlistProfile() (string, scaffold.Answers, *securitykey.Store, error) {
	var none scaffold.Answers
	configDir, err := cli.ConfigDir()
	if err != nil {
		return "", none, nil, err
	}
	name := rootFlags.Profile
	if name == "" {
		if name, err = profile.ActiveName(configDir); err != nil {
			return "", none, nil, fmt.Errorf("no active profile; pass --profile or run dotty init: %w", err)
		}
	}
	profileDir := profile.Dir(configDir, name)
	answers, err := scaffold.LoadAnswers(profileDir)
	if errors.Is(err, fs.ErrNotExist) {
		answers = scaffold.Answers{ProfileName: name}
	} else if err != nil {
		return "", none, nil, err
	}

	store, err := keyStore()
	if err != nil {
		return "", none, nil, err
	}
	return profileDir, answers, store, nil
}

// resolveSerialArgs maps serial-or-alias arguments to serials.
func resolveSerialArgs(store *securitykey.Store, args []string) ([]string, error) {
	serials := make([]string, 0, len(args))
	for _, arg := range args {
		if securitykey.IsSerial(arg) {
			serials = append(serials, arg)
			continue
		}
		serial, err := store.ResolveName(arg)
		if err != nil {
			return nil, err
		}
		serials = append(serials, serial)
	}
	return serials, nil
}

// pickAllowSerials multi-selects from the keys dotty knows: aliased ones,
// enrolled key stubs on disk, and whatever is plugged in right now (the scan
// and ykman are best-effort — either may be absent). Everything is
// preselected on a first restriction; afterwards only what the profile
// already allows.
func pickAllowSerials(ctx context.Context, ios cli.IOStreams, store *securitykey.Store,
	answers scaffold.Answers) ([]string, error) {
	known := map[string]bool{}
	for serial := range store.AliasesBySerial() {
		known[serial] = true
	}
	if dataDir, err := cli.DataDir(); err == nil {
		if refs, err := signingkey.Scan(dataDir, nil, ""); err == nil {
			for _, ref := range refs {
				known[ref.Serial] = true
			}
		}
	}
	if plugged, err := securitykey.ListSerials(ctx, newRunner(ios)); err == nil {
		for _, serial := range plugged {
			known[serial] = true
		}
	}
	if len(known) == 0 {
		return nil, errors.New("no known or connected security keys; pass the serials to allow")
	}

	current := answers.AllowedSerials
	options := make([]tui.Option, 0, len(known))
	for _, serial := range slices.Sorted(maps.Keys(known)) {
		options = append(options, tui.Option{
			Label:    securitykey.SerialLabel(store, serial),
			Value:    serial,
			Selected: len(current) == 0 || slices.Contains(current, serial),
		})
	}
	prompt := fmt.Sprintf("Allow which keys for profile %s?", answers.ProfileName)
	picked, err := tui.MultiSelect(ios, prompt, options)
	if errors.Is(err, tui.ErrNotInteractive) {
		return nil, errors.New("pass the serials or aliases to allow")
	}
	return picked, err
}

// errKeyNotAllowed reports a security key the active profile's allowlist
// rejects; callers and tests match it with errors.Is.
var errKeyNotAllowed = errors.New("not allowed for profile")

// requireAllowedSerial fails when the active profile restricts security keys
// and serial is not on its list. Callers sit on every path that uses a key:
// get, link, sign, enrollment, import, and the plugged-stub scan.
func requireAllowedSerial(serial string) error {
	profileName, allowed, err := enforcementSerials()
	if err != nil || len(allowed) == 0 {
		return err
	}
	if !slices.Contains(allowed, serial) {
		return fmt.Errorf("YubiKey %s is %w %s (see dotty security-key allow)", serial, errKeyNotAllowed, profileName)
	}
	return nil
}

// filterAllowedRefs drops stubs whose key the active profile disallows.
func filterAllowedRefs(refs []signingkey.KeyRef) ([]signingkey.KeyRef, error) {
	_, allowed, err := enforcementSerials()
	if err != nil || len(allowed) == 0 {
		return refs, err
	}
	kept := refs[:0]
	for _, ref := range refs {
		if slices.Contains(allowed, ref.Serial) {
			kept = append(kept, ref)
		}
	}
	return kept, nil
}

// filterAllowedSerials drops serials the active profile disallows.
func filterAllowedSerials(serials []string) ([]string, error) {
	_, allowed, err := enforcementSerials()
	if err != nil || len(allowed) == 0 {
		return serials, err
	}
	kept := serials[:0]
	for _, serial := range serials {
		if slices.Contains(allowed, serial) {
			kept = append(kept, serial)
		}
	}
	return kept, nil
}

// enforcementSerials returns the active profile's allowlist; a machine with
// no active profile (or a profile without answers) enforces nothing.
func enforcementSerials() (string, []string, error) {
	configDir, err := cli.ConfigDir()
	if err != nil {
		return "", nil, err
	}
	name, err := profile.ActiveName(configDir)
	if err != nil {
		return "", nil, nil // no active profile: unrestricted
	}
	answers, err := scaffold.LoadAnswers(profile.Dir(configDir, name))
	if err != nil {
		return "", nil, nil // no answers: unrestricted
	}
	return name, answers.AllowedSerials, nil
}
