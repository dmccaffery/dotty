// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"testing"
)

func TestBuildAssuan(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		keyinfo string
		want    string
	}{
		{
			name:   "fingerprint recovered from an agent-style prompt",
			prompt: "Allow use of key SHA256:abc123:more",
			want: "SETDESC Allow use of key SHA256:abc123:more\n" +
				"OPTION allow-external-password-cache\n" +
				"SETKEYINFO s/abc123\n" +
				"GETPIN\n",
		},
		{
			// ssh-keygen's agentless PIN prompt names no key; sign passes the
			// fingerprint out-of-band and it must drive the keychain cache.
			name:    "keyinfo supplied out-of-band",
			prompt:  "Enter PIN for ED25519-SK key: ",
			keyinfo: "abc123",
			want: "SETDESC Enter PIN for ED25519-SK key: \n" +
				"OPTION allow-external-password-cache\n" +
				"SETKEYINFO s/abc123\n" +
				"GETPIN\n",
		},
		{
			name:   "no key identity falls back to a plain prompt",
			prompt: "Enter your PIN",
			want:   "SETDESC Enter your PIN\nGETPIN\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAssuan(tt.prompt, tt.keyinfo); got != tt.want {
				t.Errorf("buildAssuan(%q, %q) =\n%q\nwant\n%q", tt.prompt, tt.keyinfo, got, tt.want)
			}
		})
	}
}

func TestExtractPin(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want string
	}{
		{"data line", "OK Pleased to meet you\nD 1234\nOK\n", "1234"},
		{"no data line", "OK\nERR 83886179 Operation cancelled\n", ""},
		{"empty", "", ""},
		{"bare marker", "D\n", ""},
	} {
		if got := extractPin(c.in); got != c.want {
			t.Errorf("%s: extractPin = %q, want %q", c.name, got, c.want)
		}
	}
}

// fakeAssuan records the Assuan block it was handed and returns a canned
// pinentry response.
type fakeAssuan struct {
	gotStdin string
	called   bool
	reply    string
}

func (f *fakeAssuan) RunAssuan(_ context.Context, stdin, _ string, _ ...string) (string, error) {
	f.called = true
	f.gotStdin = stdin
	return f.reply, nil
}

func TestAskPassReply(t *testing.T) {
	t.Run("user-presence prompt never invokes pinentry", func(t *testing.T) {
		f := &fakeAssuan{reply: "D 9999\n"}
		got := AskPassReply(context.Background(), f, "Confirm user presence for key ED25519-SK ...", "")
		if got != "" {
			t.Errorf("reply = %q, want empty", got)
		}
		if f.called {
			t.Error("presence path invoked pinentry")
		}
	})

	t.Run("PIN prompt forwards keyinfo and returns the entered PIN", func(t *testing.T) {
		f := &fakeAssuan{reply: "OK\nD 4242\nOK\n"}
		got := AskPassReply(context.Background(), f, "Enter PIN for ED25519-SK key: ", "fp123")
		if got != "4242" {
			t.Errorf("reply = %q, want 4242", got)
		}
		want := "SETDESC Enter PIN for ED25519-SK key: \n" +
			"OPTION allow-external-password-cache\n" +
			"SETKEYINFO s/fp123\n" +
			"GETPIN\n"
		if f.gotStdin != want {
			t.Errorf("assuan stdin =\n%q\nwant\n%q", f.gotStdin, want)
		}
	})
}
