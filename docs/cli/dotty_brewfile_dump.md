## dotty brewfile dump

Snapshot the installed brews into the Brewfile.

### Synopsis

Write the currently installed brews into the Brewfile. By default only
formulae, casks, Mac App Store apps, and Flatpaks are dumped; --all includes
every type brew bundle knows (taps, vscode, go, cargo, uv, krew, npm).
Overwriting an existing Brewfile asks first.

```
dotty brewfile dump [--all] [flags]
```

### Examples

```
  dotty brewfile dump
  dotty brewfile dump --all
```

### Options

```
      --all    dump every entry type brew bundle supports
  -h, --help   help for dump
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty brewfile](dotty_brewfile.md)	 - Manage the profile's Brewfile for reproducible brews.

