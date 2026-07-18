// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package fonts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallLobeIcons(t *testing.T) {
	font := []byte("pretend this is a ttf")
	sum := sha256.Sum256(font)

	tests := []struct {
		name          string
		body          []byte
		status        int
		preinstall    bool
		wantInstalled bool
		wantErr       bool
	}{
		{name: "downloads and installs", body: font, status: http.StatusOK, wantInstalled: true},
		{name: "already present skips download", body: font, status: http.StatusOK, preinstall: true},
		{name: "corrupt body fails checksum", body: []byte("tampered"), status: http.StatusOK, wantErr: true},
		{name: "http error", body: nil, status: http.StatusNotFound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write(tt.body)
			}))
			defer srv.Close()

			dir := filepath.Join(t.TempDir(), "fonts")
			path := filepath.Join(dir, LobeIconsFile)
			if tt.preinstall {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			old := pinnedSHA(t, hex.EncodeToString(sum[:]))
			defer old()

			installed, err := InstallLobeIcons(context.Background(), srv.Client(), srv.URL, dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("InstallLobeIcons succeeded, want error")
				}
				if _, statErr := os.Stat(path); statErr == nil {
					t.Fatal("failed install left a font on disk")
				}
				return
			}
			if err != nil {
				t.Fatalf("InstallLobeIcons: %v", err)
			}
			if installed != tt.wantInstalled {
				t.Fatalf("installed = %v, want %v", installed, tt.wantInstalled)
			}
			want := string(font)
			if tt.preinstall {
				want = "existing" // untouched
			}
			if got, err := os.ReadFile(path); err != nil || string(got) != want {
				t.Fatalf("installed font = %q, %v", got, err)
			}
		})
	}
}

// pinnedSHA swaps the package-level pin for the test body's hash and returns
// the restore func.
func pinnedSHA(t *testing.T, sum string) func() {
	t.Helper()
	old := pinSHA256
	pinSHA256 = sum
	return func() { pinSHA256 = old }
}
