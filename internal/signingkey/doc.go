// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package signingkey creates and uses SSH signing keys that live as resident
// FIDO2 credentials on hardware security keys. The on-disk artifacts are
// key-handle stubs (not private key material — the secret never leaves the
// hardware) stored per serial under $XDG_DATA_HOME/dotty/security-key, named
// id_<type>_sk_<user>. Key generation and signing shell out to ssh-keygen;
// the sign flow speaks git's gpg.ssh.program contract exactly.
package signingkey
