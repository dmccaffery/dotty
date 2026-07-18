## dotty dotfiles

Operate on the dotfiles repository dotty init generated.

### Synopsis

Link, inspect, and recover the dotfiles repository created by dotty init —
or any repository with the same layout: a home/ tree of $HOME-relative
entries plus profiles/ directories whose profile.json records the choices
that built them.

### Examples

```
  dotty dotfiles status
  dotty dotfiles link --on-conflict=backup
  dotty dotfiles restore
```

### Options

```
  -h, --help          help for dotfiles
      --repo string   dotfiles repository (default: found via $REPOS_DIR)
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty](dotty.md)	 - Utilities for a terminal-driven workflow and dotfiles.
* [dotty dotfiles link](dotty_dotfiles_link.md)	 - Symlink the repository's home tree into your home directory.
* [dotty dotfiles restore](dotty_dotfiles_restore.md)	 - Copy a backup set back over the links that displaced it.
* [dotty dotfiles status](dotty_dotfiles_status.md)	 - Report the link state of the dotfiles tree without changing it.

