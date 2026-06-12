# Hardware verification checklist

Steps that need real YubiKeys, Homebrew, or git and cannot run in CI or a sandbox. Run them on a
machine with the keys in hand after changes to the securitykey, signingkey, or brewfile areas.

macOS note: `ykman` and `fido2-token` need HID access — approve the Input Monitoring prompt on
first use or device enumeration returns nothing.

## Enumeration

- [ ] `ykman list --serials` shows every plugged-in YubiKey, one serial per line.
- [ ] `fido2-token -L` lists matching `ioreg://` devices with `vendor=0x1050`.
- [ ] `dotty signing-key list` with no keys plugged in prints the empty-state notice and exits 0.

## security-key

- [ ] `dotty security-key add` with one key plugged in uses it without prompting for a serial.
- [ ] `dotty sk add` with two keys plugged in offers the picker plus manual entry.
- [ ] Editor round-trip: accepting the description prompt opens `$EDITOR`; skipping works.
- [ ] `dotty security-key remove` shows the tree (plugged-in badge correct), collapse/expand with
      h/l, filter with /, multi-select with space, esc exits cleanly without changes.

## signing-key new

- [ ] Single key: `dotty signing-key new` enrolls without a replug prompt; ssh-keygen asks for the
      PIN on the tty; the printed pubkey matches `<stub>.pub`; stub lands under
      `~/.local/share/dotty/security-key/<serial>/` with 0700/0600 modes.
- [ ] Two keys: the replug flow identifies the re-inserted key (serial and `-O device=` path);
      enrollment lands under the correct serial.
- [ ] Two keys, esc at the replug prompt: picker appears; instructed-touch message names the chosen
      serial; touching that key enrolls it.
- [ ] `--security-key=<alias>` resolves the alias; replugging the _other_ key errors.
- [ ] Re-enrolling the same username warns about replacing the resident credential and replaces it
      only after confirmation (verify with `ykman fido credentials list`).
- [ ] Wrong PIN and unplugged-key failures surface ssh-keygen's message and a non-zero exit.

## signing-key get / list / sign

- [ ] `dotty signing-key get` prints stub + pub; `--format=key` prints one `key::sk-ssh-...` line
      with exit 0 and nothing else on stdout, including when piped (`| cat`).
- [ ] `dotty signing-key list` table filters as you type; enter prints the selected key; esc prints
      nothing and exits 0.
- [ ] `dotty signing-key sign document.txt` produces `document.txt.sig` after a PIN entry.
- [ ] With both keys plugged in, signing uses the correct one automatically (key-handle lookup).

## git integration

Dynamic mode (`dotty signing-key sign --print-git-config`, paste, then):

- [ ] `git commit -S` signs after one PIN entry; `git log --show-signature -1` verifies (with the
      allowed-signers file configured).
- [ ] `git -c gpg.ssh.defaultKeyCommand="dotty signing-key get --format=key" ...` works from a
      non-tty context (git captures stdout) — no hidden prompt, clean `key::` line.

Static mode:

- [ ] `git -c user.signingKey=<stub path> commit -S` works with `gpg.ssh.program` left default
      (plain ssh-keygen) and with it pointed at dotty.

## brewfile

- [ ] `dotty profile activate` on a fresh profile dumps a Brewfile (formulae, casks, mas, flatpak).
- [ ] `dotty brewfile add jq` appends to the Brewfile and installs.
- [ ] `dotty brewfile add <tap>/<repo>/<formula>` for an untrusted tap-qualified formula prompts,
      then `brew trust --json v1` lists it after confirmation.
- [ ] `dotty brewfile sync` with an unlisted formula installed shows the removal list and aborts
      cleanly when declined; `--force` skips the prompt.
- [ ] `dotty brewfile dump` over an existing Brewfile asks before overwriting.
- [ ] `dotty brewfile edit --sync` opens the editor, then syncs on exit.
