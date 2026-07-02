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
