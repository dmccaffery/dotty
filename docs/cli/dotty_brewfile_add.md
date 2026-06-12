## dotty brewfile add

Add brews to the Brewfile and install them.

### Synopsis

Add one or more entries to the Brewfile, then install the bundle. Entries
default to formulae; pass a type flag for anything else. Tap-qualified names
(more than one slash) of formulae, casks, and taps go through Homebrew's
trust gate first — dotty asks before trusting anything new.

```
dotty brewfile add [--tap | --cask | --formula | ...] <name> [...] [flags]
```

### Examples

```
  dotty brewfile add ripgrep jq
  dotty brewfile add --cask ghostty
  dotty brewfile add --tap fluxcd/tap
  dotty brewfile add acme/tap/widget
```

### Options

```
      --cargo     add Cargo packages
      --cask      add casks
      --flatpak   add Flatpak packages
      --formula   add formulae (the default)
      --go        add Go packages
  -h, --help      help for add
      --krew      add Krew plugins
      --npm       add npm packages
      --tap       add taps
      --uv        add uv tools
      --vscode    add VSCode extensions
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty brewfile](dotty_brewfile.md)	 - Manage the profile's Brewfile for reproducible brews.

