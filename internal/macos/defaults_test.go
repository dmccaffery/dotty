// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package macos

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fakeRunner records commands instead of running them.
type fakeRunner struct {
	commands [][]string
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	f.commands = append(f.commands, append([]string{name}, args...))
	return nil, nil
}

func (f *fakeRunner) RunInteractive(_ context.Context, name string, args ...string) error {
	f.commands = append(f.commands, append([]string{name}, args...))
	return nil
}

func TestApply(t *testing.T) {
	r := &fakeRunner{}
	if err := Apply(context.Background(), r, []string{"dock", "spaces", "keyboard"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var writes, restarts int
	for _, c := range r.commands {
		switch c[0] {
		case "defaults":
			if c[1] != "write" {
				t.Errorf("non-write defaults command: %v", c)
			}
			writes++
		case "killall":
			restarts++
			if c[1] != "Dock" {
				t.Errorf("unexpected restart: %v", c)
			}
		default:
			t.Errorf("unexpected command: %v", c)
		}
	}
	if writes != 7 { // dock 4 + spaces 2 + keyboard 1
		t.Errorf("writes = %d, want 7", writes)
	}
	if restarts != 1 { // dock and spaces share the Dock restart
		t.Errorf("Dock restarted %d times, want once", restarts)
	}
}

func TestApplyRejectsUnknownGroup(t *testing.T) {
	r := &fakeRunner{}
	if err := Apply(context.Background(), r, []string{"nope"}); err == nil {
		t.Fatal("Apply accepted an unknown group")
	}
	if len(r.commands) != 0 {
		t.Fatalf("commands ran despite the error: %v", r.commands)
	}
}

func TestEnforcePIVUsesSudoInteractively(t *testing.T) {
	r := &fakeRunner{}
	if err := EnforcePIV(context.Background(), r); err != nil {
		t.Fatalf("EnforcePIV: %v", err)
	}
	if len(r.commands) != 2 || r.commands[0][0] != "sudo" || r.commands[1][0] != "sudo" {
		t.Fatalf("commands = %v, want two sudo invocations", r.commands)
	}
	if !slices.Contains(r.commands[0], "enforceSmartCard") {
		t.Errorf("first command misses enforceSmartCard: %v", r.commands[0])
	}
}

func TestSetWallpaper(t *testing.T) {
	r := &fakeRunner{}
	if err := SetWallpaper(context.Background(), r, "/pics/beach.png"); err != nil {
		t.Fatalf("SetWallpaper: %v", err)
	}
	if len(r.commands) != 1 || r.commands[0][0] != "osascript" ||
		!strings.Contains(r.commands[0][2], `"/pics/beach.png"`) {
		t.Fatalf("commands = %v", r.commands)
	}
}

func TestWallpapers(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"beach.png", "city.JPG", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := Wallpapers(dir)
	want := []string{"beach.png", "city.JPG"}
	if !slices.Equal(got, want) {
		t.Fatalf("Wallpapers = %v, want %v", got, want)
	}
	if Wallpapers(filepath.Join(dir, "absent")) != nil {
		t.Fatal("missing dir should list nothing")
	}
}
