// MIT License
//
// Copyright (c) 2026 Bitwise Media Group
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bitwise-media-group/dotty/internal/securitykey"
)

// execDotty runs the real command tree, as a user invocation would.
func execDotty(t *testing.T, args ...string) error {
	t.Helper()
	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(context.Background())
}

// TestFlagBeforeVerb pins DESIGN's ergonomics: noun-level persistent flags
// parse before or after the verb, with '=' or space separators, by running
// `security-key add` end-to-end against a scratch data dir.
func TestFlagBeforeVerb(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "flag before verb, equals form", args: []string{"security-key", "--serial=111", "add", "--name=k1"}},
		{name: "flag before verb, space form", args: []string{"security-key", "--serial", "222", "add", "--name", "k2"}},
		{name: "flag after verb", args: []string{"security-key", "add", "--serial=333", "--name=k3"}},
		{name: "noun alias sk", args: []string{"sk", "--serial=444", "add", "--name=k4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			t.Setenv("XDG_DATA_HOME", dataDir)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			if err := execDotty(t, tt.args...); err != nil {
				t.Fatalf("execute %v: %v", tt.args, err)
			}

			store, err := securitykey.LoadStore(securitykey.StorePath(filepath.Join(dataDir, "dotty")))
			if err != nil {
				t.Fatal(err)
			}
			if names := store.Names(); len(names) != 1 {
				t.Errorf("store names = %v, want exactly one alias", names)
			}
			// Reset the persistent flag value for the next case.
			securityKeyFlags.Serial = ""
			securityKeyAddFlags.Name = ""
		})
	}
}

// TestDispatchArgs pins the git-signing argv rewrite: gpg.ssh.program may be
// the dotty binary itself (git passes -Y first) or a dotty-ssh-sign symlink.
func TestDispatchArgs(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "plain invocation untouched",
			argv: []string{"/usr/local/bin/dotty", "profile", "new"},
			want: []string{"profile", "new"},
		},
		{
			name: "git -Y invocation rewritten",
			argv: []string{"/usr/local/bin/dotty", "-Y", "sign", "-n", "git", "-f", "/k", "/buf"},
			want: []string{"signing-key", "sign", "-Y", "sign", "-n", "git", "-f", "/k", "/buf"},
		},
		{
			name: "shim symlink rewritten",
			argv: []string{"/Users/x/.local/bin/dotty-ssh-sign", "-Y", "sign", "-f", "/k", "/buf"},
			want: []string{"signing-key", "sign", "-Y", "sign", "-f", "/k", "/buf"},
		},
		{
			name: "bare invocation untouched",
			argv: []string{"dotty"},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dispatchArgs(tt.argv); !slices.Equal(got, tt.want) {
				t.Errorf("dispatchArgs(%v) = %v, want %v", tt.argv, got, tt.want)
			}
		})
	}
}
