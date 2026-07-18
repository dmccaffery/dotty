## dotty dotfiles restore

Copy a backup set back over the links that displaced it.

### Synopsis

Restore the files a link run backed up: every file in the chosen backup set
is copied back to the absolute path it came from, replacing the symlink that
displaced it. Without --timestamp, an interactive picklist offers the
available sets, newest first. The set is copied, not consumed — a restore can
be repeated.

```
dotty dotfiles restore [flags]
```

### Examples

```
  dotty dotfiles restore
  dotty dotfiles restore --timestamp=2026-07-15T10-30-00
```

### Options

```
  -h, --help               help for restore
      --timestamp string   backup set to restore (a directory name under the backups dir)
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
      --repo string      dotfiles repository (default: found via $REPOS_DIR)
```

### SEE ALSO

* [dotty dotfiles](dotty_dotfiles.md)	 - Operate on the dotfiles repository dotty init generated.

