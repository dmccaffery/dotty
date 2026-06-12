## dotty brewfile edit

Open the Brewfile in the default editor.

### Synopsis

Open the Brewfile in $VISUAL / $EDITOR. With --sync or --upgrade, the
corresponding command runs after the editor exits.

```
dotty brewfile edit [--sync | --upgrade] [flags]
```

### Examples

```
  dotty brewfile edit
  dotty brewfile edit --sync
```

### Options

```
  -h, --help                             help for edit
      --sync dotty brewfile sync         run dotty brewfile sync after editing
      --upgrade dotty brewfile upgrade   run dotty brewfile upgrade after editing
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty brewfile](dotty_brewfile.md)	 - Manage the profile's Brewfile for reproducible brews.

