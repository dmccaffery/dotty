// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package securitykey

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	s, err := LoadStore(StorePath(t.TempDir()))
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	return s
}

func TestStoreAdd(t *testing.T) {
	t.Run("round-trips through disk", func(t *testing.T) {
		path := StorePath(t.TempDir())
		s, err := LoadStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Add("12345678", "work", "main key"); err != nil {
			t.Fatalf("Add() error: %v", err)
		}
		if err := s.Save(); err != nil {
			t.Fatalf("Save() error: %v", err)
		}

		reloaded, err := LoadStore(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		serial, err := reloaded.ResolveName("work")
		if err != nil || serial != "12345678" {
			t.Errorf("ResolveName(work) = %q, %v", serial, err)
		}
		aliases := reloaded.AliasesBySerial()["12345678"]
		if len(aliases) != 1 || aliases[0].Description != "main key" {
			t.Errorf("aliases = %+v", aliases)
		}
		if aliases[0].CreatedAt.IsZero() {
			t.Error("CreatedAt is zero")
		}
	})

	t.Run("names are unique across serials", func(t *testing.T) {
		s := tempStore(t)
		if err := s.Add("111", "work", ""); err != nil {
			t.Fatal(err)
		}
		err := s.Add("222", "work", "")
		if !errors.Is(err, ErrNameTaken) {
			t.Errorf("error = %v, want ErrNameTaken", err)
		}
	})

	t.Run("multiple aliases per serial", func(t *testing.T) {
		s := tempStore(t)
		for _, name := range []string{"work", "primary"} {
			if err := s.Add("111", name, ""); err != nil {
				t.Fatal(err)
			}
		}
		if got := len(s.AliasesBySerial()["111"]); got != 2 {
			t.Errorf("aliases = %d, want 2", got)
		}
		// Sorted within the serial.
		aliases := s.AliasesBySerial()["111"]
		if aliases[0].Name != "primary" || aliases[1].Name != "work" {
			t.Errorf("order = %s, %s", aliases[0].Name, aliases[1].Name)
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		s := tempStore(t)
		if err := s.Add("111", "123", ""); err == nil {
			t.Error("all-digit alias accepted")
		}
		if err := s.Add("abc", "work", ""); err == nil {
			t.Error("non-numeric serial accepted")
		}
	})
}

func TestStoreRemove(t *testing.T) {
	s := tempStore(t)
	for serial, name := range map[string]string{"111": "work", "222": "backup"} {
		if err := s.Add(serial, name, ""); err != nil {
			t.Fatal(err)
		}
	}
	if removed := s.Remove("work", "ghost"); removed != 1 {
		t.Errorf("Remove() = %d, want 1", removed)
	}
	if _, err := s.ResolveName("work"); !errors.Is(err, ErrUnknownAlias) {
		t.Error("removed alias still resolves")
	}
	// The empty serial entry is gone from the grouping.
	if _, ok := s.AliasesBySerial()["111"]; ok {
		t.Error("serial 111 still present after its last alias was removed")
	}
	if names := s.Names(); len(names) != 1 || names[0] != "backup" {
		t.Errorf("Names() = %v", names)
	}
}

func TestLoadStore(t *testing.T) {
	t.Run("missing file is an empty store", func(t *testing.T) {
		s, err := LoadStore(filepath.Join(t.TempDir(), "nope", "aliases.json"))
		if err != nil {
			t.Fatalf("LoadStore() error: %v", err)
		}
		if len(s.Names()) != 0 {
			t.Errorf("Names() = %v, want empty", s.Names())
		}
	})

	t.Run("corrupt JSON is a hard error naming the file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "aliases.json")
		if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadStore(path)
		if err == nil {
			t.Fatal("LoadStore(corrupt) error = nil")
		}
	})

	t.Run("duplicate names across serials are rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "aliases.json")
		doc := `{"version":1,"keys":{"111":{"aliases":[{"name":"work"}]},"222":{"aliases":[{"name":"work"}]}}}`
		if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadStore(path); err == nil {
			t.Fatal("LoadStore(duplicates) error = nil")
		}
	})

	t.Run("save writes repo-friendly permissions", func(t *testing.T) {
		profileDir := t.TempDir()
		s, err := LoadStore(StorePath(profileDir))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Add("111", "work", ""); err != nil {
			t.Fatal(err)
		}
		if err := s.Save(); err != nil {
			t.Fatal(err)
		}
		fileInfo, err := os.Stat(StorePath(profileDir))
		if err != nil {
			t.Fatal(err)
		}
		if perm := fileInfo.Mode().Perm(); perm != 0o644 {
			t.Errorf("file perm = %o, want 644", perm)
		}
	})
}
