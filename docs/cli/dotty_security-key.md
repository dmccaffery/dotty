## dotty security-key

Manage hardware security keys.

### Synopsis

Name your hardware security keys: aliases map memorable names to YubiKey
serial numbers, so other commands can say --security-key=work instead of a
serial. Aliases live in the private dotty data directory
($XDG_DATA_HOME/dotty/security-key), not in shareable config.

### Examples

```
  dotty security-key add --name=work
  dotty sk --serial=12345678 add --name=backup
  dotty security-key remove
```

### Options

```
  -h, --help            help for security-key
      --serial string   serial number of the security key
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty](dotty.md)	 - Utilities for a terminal-driven workflow and dotfiles.
* [dotty security-key add](dotty_security-key_add.md)	 - Add a named alias for a security key.
* [dotty security-key remove](dotty_security-key_remove.md)	 - Remove security key aliases.

