## dotty brewfile sync

Make the machine match the Brewfile exactly.

### Synopsis

Synchronise the machine with the Brewfile — install what's listed, upgrade
what's outdated, and remove (zap) what isn't listed. When anything would be
removed, dotty shows the list and asks first unless --force is set.

```
dotty brewfile sync [--force] [flags]
```

### Examples

```
  dotty brewfile sync
  dotty brewfile sync --force
```

### Options

```
      --force   remove unlisted brews without asking
  -h, --help    help for sync
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty brewfile](dotty_brewfile.md)	 - Manage the profile's Brewfile for reproducible brews.

