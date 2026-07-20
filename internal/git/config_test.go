// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestConfigLookup(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a set value", func(t *testing.T) {
		var gotArgs []string
		r := &fakeRunner{output: func(args []string) ([]byte, error) {
			gotArgs = args
			return []byte("deavon@example.com\n"), nil
		}}
		value, found, err := ConfigLookup(ctx, r, "user.email")
		if err != nil {
			t.Fatalf("ConfigLookup() error: %v", err)
		}
		if !found || value != "deavon@example.com" {
			t.Errorf("ConfigLookup() = %q, %v; want deavon@example.com, true", value, found)
		}
		// --default "" is what lets the unset case return cleanly.
		want := []string{"config", "--default", "", "--get", "user.email"}
		if !slices.Equal(gotArgs, want) {
			t.Errorf("git args = %v, want %v", gotArgs, want)
		}
	})

	t.Run("empty output reads as unset", func(t *testing.T) {
		r := &fakeRunner{output: func([]string) ([]byte, error) { return []byte("\n"), nil }}
		value, found, err := ConfigLookup(ctx, r, "gpg.ssh.allowedSignersFile")
		if err != nil {
			t.Fatalf("ConfigLookup() error: %v", err)
		}
		if found || value != "" {
			t.Errorf("ConfigLookup() = %q, %v; want \"\", false", value, found)
		}
	})

	t.Run("propagates a genuine git failure", func(t *testing.T) {
		sentinel := errors.New("git exploded")
		r := &fakeRunner{output: func([]string) ([]byte, error) { return nil, sentinel }}
		if _, _, err := ConfigLookup(ctx, r, "user.email"); !errors.Is(err, sentinel) {
			t.Errorf("ConfigLookup() error = %v, want wrapped %v", err, sentinel)
		}
	})
}

// TestConfigLookupBool pins the two-lookup canonicalization: the raw read
// detects a stored value, the --type=bool read canonicalizes it, and the
// valueless `[section] key` form — raw output empty, typed output true —
// reads as set-to-true rather than unset.
func TestConfigLookupBool(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("bad boolean config value 'banana'")

	cases := []struct {
		name      string
		raw       string // output of the raw --get
		typed     string // output of the --type=bool --get
		typedErr  error
		wantValue string
		wantFound bool
		wantErr   bool
	}{
		{"yes canonicalizes to true", "yes\n", "true\n", nil, "true", true, false},
		{"off canonicalizes to false", "off\n", "false\n", nil, "false", true, false},
		{"valueless key reads true", "\n", "true\n", nil, "true", true, false},
		{"unset key", "\n", "false\n", nil, "", false, false},
		{"unreadable value errors", "banana\n", "", sentinel, "", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &fakeRunner{output: func(args []string) ([]byte, error) {
				if slices.Contains(args, "--type=bool") {
					return []byte(c.typed), c.typedErr
				}
				return []byte(c.raw), nil
			}}
			value, found, err := ConfigLookupBool(ctx, r, "dotty.propose.browse")
			if (err != nil) != c.wantErr {
				t.Fatalf("ConfigLookupBool() error = %v, wantErr %v", err, c.wantErr)
			}
			if value != c.wantValue || found != c.wantFound {
				t.Errorf("ConfigLookupBool() = %q, %v; want %q, %v", value, found, c.wantValue, c.wantFound)
			}
		})
	}
}
