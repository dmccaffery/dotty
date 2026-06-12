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

package signingkey

import (
	"context"
	"slices"
	"testing"
)

type fakeInteractive struct {
	name string
	args []string
	err  error
}

func (f *fakeInteractive) RunInteractive(_ context.Context, name string, args ...string) error {
	f.name = name
	f.args = args
	return f.err
}

// TestNewKeyArgs is the regression table for DESIGN's exact ssh-keygen
// invocation — any drift in the -O set is a bug.
func TestNewKeyArgs(t *testing.T) {
	t.Run("without device", func(t *testing.T) {
		got := NewKeyArgs(KeygenOptions{Type: "ed25519", User: "deavon", Path: "/d/sk/111/id_ed25519_sk_deavon"})
		want := []string{
			"-t", "ed25519-sk",
			"-f", "/d/sk/111/id_ed25519_sk_deavon",
			"-O", "resident",
			"-O", "verify-required",
			"-O", "no-touch-required",
			"-O", "application=ssh:deavon",
			"-O", "user=deavon",
			"-C", "deavon",
			"-N", "",
		}
		if !slices.Equal(got, want) {
			t.Errorf("args =\n%v\nwant\n%v", got, want)
		}
	})

	t.Run("with device", func(t *testing.T) {
		got := NewKeyArgs(KeygenOptions{Type: "ecdsa", User: "u", Path: "/p", Device: "ioreg://42"})
		if !slices.Contains(got, "device=ioreg://42") {
			t.Errorf("args %v missing device option", got)
		}
		if got[1] != "ecdsa-sk" {
			t.Errorf("type arg = %q, want ecdsa-sk", got[1])
		}
	})
}

func TestGenerate(t *testing.T) {
	ctx := context.Background()

	t.Run("invokes ssh-keygen", func(t *testing.T) {
		f := &fakeInteractive{}
		err := Generate(ctx, f, KeygenOptions{Type: "ed25519", User: "deavon", Path: "/p"})
		if err != nil {
			t.Fatalf("Generate() error: %v", err)
		}
		if f.name != "ssh-keygen" {
			t.Errorf("ran %q, want ssh-keygen", f.name)
		}
	})

	t.Run("rejects bad type including the DESIGN typo", func(t *testing.T) {
		for _, typ := range []string{"edcsa", "rsa", ""} {
			if err := Generate(ctx, &fakeInteractive{}, KeygenOptions{Type: typ, User: "u", Path: "/p"}); err == nil {
				t.Errorf("Generate(type=%q) error = nil", typ)
			}
		}
	})

	t.Run("rejects bad usernames", func(t *testing.T) {
		for _, user := range []string{"", "a b", "a/b"} {
			if err := Generate(ctx, &fakeInteractive{}, KeygenOptions{Type: "ed25519", User: user, Path: "/p"}); err == nil {
				t.Errorf("Generate(user=%q) error = nil", user)
			}
		}
	})
}
