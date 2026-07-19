// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"io/fs"
	"testing"
)

// helloFP is base64(sha256("hello")) — the fingerprint of the blob "aGVsbG8="
// used as the .pub key material in these tests.
const helloFP = "LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ"

// readFixture serves id_sk_current.pub and fails every other path, standing in
// for os.ReadFile.
func readFixture(path string) ([]byte, error) {
	if path != "/home/u/.ssh/id_sk_current.pub" {
		return nil, fs.ErrNotExist
	}
	return []byte("sk-ssh-ed25519@openssh.com aGVsbG8= u@host\n"), nil
}

func TestBuildAssuan(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		keyinfo string
		want    string
	}{
		{
			name:    "keyinfo enables the keychain cache",
			prompt:  "Enter PIN for ED25519-SK key: ",
			keyinfo: "abc123",
			want: "SETDESC Enter PIN for ED25519-SK key: \n" +
				"OPTION allow-external-password-cache\n" +
				"SETKEYINFO s/abc123\n" +
				"GETPIN\n",
		},
		{
			name:   "no keyinfo falls back to a plain prompt",
			prompt: "Enter your PIN",
			want:   "SETDESC Enter your PIN\nGETPIN\n",
		},
		{
			// A multi-line prompt must not terminate SETDESC early — Assuan
			// treats a raw newline as the end of the command.
			name:   "prompt newlines and percents are Assuan-escaped",
			prompt: "line one\nline two 100%",
			want:   "SETDESC line one%0Aline two 100%25\nGETPIN\n",
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

func TestResolveKeyInfo(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		keyinfo string
		want    string
	}{
		{
			// ssh-keygen's agentless PIN prompt names no key; sign passes the
			// fingerprint out-of-band and it must drive the keychain cache.
			name:    "out-of-band keyinfo wins",
			prompt:  "Enter PIN for ED25519-SK key: ",
			keyinfo: "abc123",
			want:    "abc123",
		},
		{
			name:   "fingerprint recovered from an agent-style prompt",
			prompt: "Allow use of key SHA256:abc123:more",
			want:   "abc123",
		},
		{
			// ssh's client-auth prompt names the key by file path; the .pub
			// sidecar is fingerprinted so the keychain item the sign path
			// created is shared.
			name:   "client-auth prompt fingerprints the named key file",
			prompt: "Enter PIN for ED25519-SK key /home/u/.ssh/id_sk_current: ",
			want:   helloFP,
		},
		{
			name:   "unreadable sidecar yields no keyinfo",
			prompt: "Enter PIN for ED25519-SK key /home/u/.ssh/id_missing: ",
			want:   "",
		},
		{
			name:   "unidentifiable prompt yields no keyinfo",
			prompt: "Enter your PIN",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveKeyInfo(tt.prompt, tt.keyinfo, readFixture); got != tt.want {
				t.Errorf("resolveKeyInfo(%q, %q) = %q, want %q", tt.prompt, tt.keyinfo, got, tt.want)
			}
		})
	}
}

func TestPromptKeyPath(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{"client-auth prompt", "Enter PIN for ED25519-SK key /home/u/.ssh/id_sk_current: ", "/home/u/.ssh/id_sk_current"},
		{"no trailing space after the colon", "Enter PIN for ECDSA-SK key /tmp/k:", "/tmp/k"},
		{"agentless prompt names no key", "Enter PIN for ED25519-SK key: ", ""},
		{"unrelated prompt", "Confirm user presence for key ED25519-SK", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := promptKeyPath(tt.prompt); got != tt.want {
				t.Errorf("promptKeyPath(%q) = %q, want %q", tt.prompt, got, tt.want)
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
	err      error
}

func (f *fakeAssuan) RunAssuan(_ context.Context, stdin, _ string, _ ...string) (string, error) {
	f.called = true
	f.gotStdin = stdin
	return f.reply, f.err
}

// hostAuthPrompt is the multi-line echo prompt ssh emits for an unknown host
// key; it carries no $SSH_ASKPASS_PROMPT hint.
const hostAuthPrompt = "The authenticity of host 'h (10.0.0.1)' can't be established.\n" +
	"ED25519 key fingerprint is SHA256:abc.\n" +
	"Are you sure you want to continue connecting (yes/no/[fingerprint])?"

func TestAskPassReply(t *testing.T) {
	t.Run("user-presence prompt never invokes pinentry", func(t *testing.T) {
		f := &fakeAssuan{reply: "D 9999\n"}
		got := AskPassReply(context.Background(), f, "Confirm user presence for key ED25519-SK ...", "", "", readFixture)
		if got != "" {
			t.Errorf("reply = %q, want empty", got)
		}
		if f.called {
			t.Error("presence path invoked pinentry")
		}
	})

	t.Run("display-only hint never invokes pinentry", func(t *testing.T) {
		f := &fakeAssuan{reply: "D 9999\n"}
		got := AskPassReply(context.Background(), f, "Touch your security key", "none", "", readFixture)
		if got != "" {
			t.Errorf("reply = %q, want empty", got)
		}
		if f.called {
			t.Error("display-only path invoked pinentry")
		}
	})

	t.Run("PIN prompt forwards keyinfo and returns the entered PIN", func(t *testing.T) {
		f := &fakeAssuan{reply: "OK\nD 4242\nOK\n"}
		got := AskPassReply(context.Background(), f, "Enter PIN for ED25519-SK key: ", "", "fp123", readFixture)
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

	t.Run("client-auth prompt caches against the named key's fingerprint", func(t *testing.T) {
		f := &fakeAssuan{reply: "OK\nD 4242\nOK\n"}
		prompt := "Enter PIN for ED25519-SK key /home/u/.ssh/id_sk_current: "
		got := AskPassReply(context.Background(), f, prompt, "", "", readFixture)
		if got != "4242" {
			t.Errorf("reply = %q, want 4242", got)
		}
		want := "SETDESC " + prompt + "\n" +
			"OPTION allow-external-password-cache\n" +
			"SETKEYINFO s/" + helloFP + "\n" +
			"GETPIN\n"
		if f.gotStdin != want {
			t.Errorf("assuan stdin =\n%q\nwant\n%q", f.gotStdin, want)
		}
	})

	t.Run("host-authenticity prompt becomes a CONFIRM, not a GETPIN", func(t *testing.T) {
		f := &fakeAssuan{reply: "OK Pleased to meet you\nOK\nOK\n"}
		got := AskPassReply(context.Background(), f, hostAuthPrompt, "", "", readFixture)
		if got != "yes" {
			t.Errorf("reply = %q, want yes", got)
		}
		want := "SETDESC The authenticity of host 'h (10.0.0.1)' can't be established.%0A" +
			"ED25519 key fingerprint is SHA256:abc.%0A" +
			"Are you sure you want to continue connecting (yes/no/[fingerprint])?\n" +
			"CONFIRM\n"
		if f.gotStdin != want {
			t.Errorf("assuan stdin =\n%q\nwant\n%q", f.gotStdin, want)
		}
	})

	t.Run("cancelled confirmation answers no", func(t *testing.T) {
		f := &fakeAssuan{reply: "OK Pleased to meet you\nOK\nERR 83886179 Operation cancelled\n"}
		if got := AskPassReply(context.Background(), f, hostAuthPrompt, "", "", readFixture); got != "no" {
			t.Errorf("reply = %q, want no", got)
		}
	})

	t.Run("pinentry failure answers no", func(t *testing.T) {
		f := &fakeAssuan{err: fs.ErrNotExist}
		if got := AskPassReply(context.Background(), f, hostAuthPrompt, "", "", readFixture); got != "no" {
			t.Errorf("reply = %q, want no", got)
		}
	})

	t.Run("confirm hint overrides secret handling", func(t *testing.T) {
		f := &fakeAssuan{reply: "OK\nOK\nOK\n"}
		got := AskPassReply(context.Background(), f, "Allow use of key foo?", "confirm", "", readFixture)
		if got != "yes" {
			t.Errorf("reply = %q, want yes", got)
		}
		if want := "SETDESC Allow use of key foo?\nCONFIRM\n"; f.gotStdin != want {
			t.Errorf("assuan stdin =\n%q\nwant\n%q", f.gotStdin, want)
		}
	})
}
