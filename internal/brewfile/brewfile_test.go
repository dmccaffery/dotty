// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package brewfile

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// fakeRunner records argv and serves scripted results. Output results are
// consumed in call order; Run errors likewise.
type fakeRunner struct {
	calls      [][]string
	outputs    [][]byte
	outputErrs []error
	runErrs    []error
	outputN    int
	runN       int
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	var out []byte
	var err error
	if f.outputN < len(f.outputs) {
		out = f.outputs[f.outputN]
	}
	if f.outputN < len(f.outputErrs) {
		err = f.outputErrs[f.outputN]
	}
	f.outputN++
	return out, err
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	f.calls = append(f.calls, append([]string{name}, args...))
	var err error
	if f.runN < len(f.runErrs) {
		err = f.runErrs[f.runN]
	}
	f.runN++
	return err
}

func (f *fakeRunner) argv(i int) string { return strings.Join(f.calls[i], " ") }

func yes(string) (bool, error) { return true, nil }
func no(string) (bool, error)  { return false, nil }

const emptyTrust = `{"taps":[],"formulae":[],"casks":[],"commands":[]}`

func TestAdd(t *testing.T) {
	ctx := context.Background()

	t.Run("plain formula skips trust and adds then installs", func(t *testing.T) {
		r := &fakeRunner{}
		if err := Add(ctx, r, "/p/Brewfile", KindFormula, []string{"jq", "ripgrep"}, no); err != nil {
			t.Fatalf("Add() error: %v", err)
		}
		want := []string{
			"brew bundle add --file=/p/Brewfile --formula jq ripgrep",
			"brew bundle install --file=/p/Brewfile",
		}
		if len(r.calls) != len(want) {
			t.Fatalf("calls = %v", r.calls)
		}
		for i, w := range want {
			if r.argv(i) != w {
				t.Errorf("call %d = %q, want %q", i, r.argv(i), w)
			}
		}
	})

	t.Run("untrusted tap-qualified formula is trusted after confirmation", func(t *testing.T) {
		r := &fakeRunner{outputs: [][]byte{[]byte(emptyTrust)}}
		if err := Add(ctx, r, "/p/Brewfile", KindFormula, []string{"acme/tap/widget"}, yes); err != nil {
			t.Fatalf("Add() error: %v", err)
		}
		want := []string{
			"brew trust --json v1",
			"brew trust --formula acme/tap/widget",
			"brew bundle add --file=/p/Brewfile --formula acme/tap/widget",
			"brew bundle install --file=/p/Brewfile",
		}
		for i, w := range want {
			if r.argv(i) != w {
				t.Errorf("call %d = %q, want %q", i, r.argv(i), w)
			}
		}
	})

	t.Run("already-trusted name skips the trust write", func(t *testing.T) {
		trusted := `{"taps":[],"formulae":["acme/tap/widget"],"casks":[],"commands":[]}`
		r := &fakeRunner{outputs: [][]byte{[]byte(trusted)}}
		if err := Add(ctx, r, "/p/Brewfile", KindFormula, []string{"acme/tap/widget"}, no); err != nil {
			t.Fatalf("Add() error: %v", err)
		}
		for _, call := range r.calls {
			if slices.Contains(call, "--formula") && call[1] == "trust" {
				t.Errorf("unexpected trust write: %v", call)
			}
		}
	})

	t.Run("declined trust aborts before any write", func(t *testing.T) {
		r := &fakeRunner{outputs: [][]byte{[]byte(emptyTrust)}}
		err := Add(ctx, r, "/p/Brewfile", KindCask, []string{"acme/tap/widget"}, no)
		if err == nil {
			t.Fatal("Add() error = nil, want declined-trust failure")
		}
		if len(r.calls) != 1 { // only the trust-store read
			t.Errorf("calls = %v, want only the trust read", r.calls)
		}
	})

	t.Run("non-trustable kinds never consult trust", func(t *testing.T) {
		r := &fakeRunner{}
		if err := Add(ctx, r, "/p/Brewfile", KindNPM, []string{"acme/scope/pkg"}, no); err != nil {
			t.Fatalf("Add() error: %v", err)
		}
		if r.argv(0) != "brew bundle add --file=/p/Brewfile --npm acme/scope/pkg" {
			t.Errorf("call 0 = %q", r.argv(0))
		}
	})
}

func TestUpgrade(t *testing.T) {
	r := &fakeRunner{}
	if err := Upgrade(context.Background(), r, "/p/Brewfile"); err != nil {
		t.Fatalf("Upgrade() error: %v", err)
	}
	if want := "brew bundle install --file=/p/Brewfile --upgrade"; r.argv(0) != want {
		t.Errorf("argv = %q, want %q", r.argv(0), want)
	}
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	installArgv := "brew bundle install --file=/p/Brewfile --force --force-cleanup --upgrade --zap"

	t.Run("force skips the dry-run", func(t *testing.T) {
		r := &fakeRunner{}
		if err := Sync(ctx, r, "/p/Brewfile", true, nil); err != nil {
			t.Fatalf("Sync() error: %v", err)
		}
		if len(r.calls) != 1 || r.argv(0) != installArgv {
			t.Errorf("calls = %v", r.calls)
		}
	})

	t.Run("clean dry-run proceeds without confirmation", func(t *testing.T) {
		r := &fakeRunner{}
		confirmed := false
		err := Sync(ctx, r, "/p/Brewfile", false, func([]string) (bool, error) { confirmed = true; return true, nil })
		if err != nil {
			t.Fatalf("Sync() error: %v", err)
		}
		if confirmed {
			t.Error("confirm called with nothing to remove")
		}
		if r.argv(0) != "brew bundle cleanup --file=/p/Brewfile" {
			t.Errorf("dry-run argv = %q", r.argv(0))
		}
		if r.argv(1) != installArgv {
			t.Errorf("install argv = %q", r.argv(1))
		}
	})

	t.Run("removals are confirmed before installing", func(t *testing.T) {
		r := &fakeRunner{
			outputs:    [][]byte{[]byte("Would uninstall formulae:\njq\n")},
			outputErrs: []error{errors.New("exit status 1")},
		}
		var got []string
		err := Sync(ctx, r, "/p/Brewfile", false, func(removals []string) (bool, error) {
			got = removals
			return true, nil
		})
		if err != nil {
			t.Fatalf("Sync() error: %v", err)
		}
		if !slices.Contains(got, "jq") {
			t.Errorf("removals = %v, want to contain jq", got)
		}
		if r.argv(len(r.calls)-1) != installArgv {
			t.Errorf("last call = %q, want install", r.argv(len(r.calls)-1))
		}
	})

	t.Run("declining the removals aborts", func(t *testing.T) {
		r := &fakeRunner{
			outputs:    [][]byte{[]byte("Would uninstall formulae:\njq\n")},
			outputErrs: []error{errors.New("exit status 1")},
		}
		if err := Sync(ctx, r, "/p/Brewfile", false, func([]string) (bool, error) { return false, nil }); err != nil {
			t.Fatalf("Sync() error: %v", err)
		}
		if len(r.calls) != 1 {
			t.Errorf("calls = %v, want dry-run only", r.calls)
		}
	})

	t.Run("dry-run failure with no output is a real error", func(t *testing.T) {
		r := &fakeRunner{outputErrs: []error{errors.New("brew exploded")}}
		if err := Sync(ctx, r, "/p/Brewfile", false, nil); err == nil {
			t.Fatal("Sync() error = nil, want failure")
		}
	})
}

func TestDump(t *testing.T) {
	ctx := context.Background()

	t.Run("default kinds per DESIGN", func(t *testing.T) {
		r := &fakeRunner{}
		if err := Dump(ctx, r, "/p/Brewfile", false, false); err != nil {
			t.Fatalf("Dump() error: %v", err)
		}
		want := "brew bundle dump --file=/p/Brewfile --formula --cask --mas --flatpak"
		if r.argv(0) != want {
			t.Errorf("argv = %q, want %q", r.argv(0), want)
		}
	})

	t.Run("all spells out every type flag", func(t *testing.T) {
		r := &fakeRunner{}
		if err := Dump(ctx, r, "/p/Brewfile", true, true); err != nil {
			t.Fatalf("Dump() error: %v", err)
		}
		argv := r.argv(0)
		for _, flag := range []string{"--tap", "--vscode", "--go", "--cargo", "--uv", "--krew", "--npm", "--force"} {
			if !strings.Contains(argv, flag) {
				t.Errorf("argv %q missing %s", argv, flag)
			}
		}
	})
}

func TestKindFlags(t *testing.T) {
	for kind, want := range map[Kind]string{KindFormula: "--formula", KindFlatpak: "--flatpak", "mas": "--mas"} {
		if got := kind.flag(); got != want {
			t.Errorf("flag(%s) = %q, want %q", kind, got, want)
		}
	}
}
