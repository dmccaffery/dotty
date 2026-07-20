## dotty dotfiles link

Symlink the repository's home tree into your home directory.

### Synopsis

Link the repository's home/ tree over $HOME: whole files and directories
are linked folded, existing real directories are descended into, and stale
symlinks are replaced. A real file in the way is resolved per --on-conflict;
link defaults to fail so an unexpected file stops a re-link instead of being
moved. Legacy files that shadow the rendered configuration from outside any
link site (~/.gitconfig, ~/.zshrc and the other bare zsh startup files) are
backed up and removed; restore them with dotty dotfiles restore.

```
dotty dotfiles link [flags]
```

### Examples

```
  dotty dotfiles link
  dotty dotfiles link --repo ~/Repos/dotfiles --on-conflict=backup
```

### Options

```
  -h, --help                 help for link
      --on-conflict string   existing-file resolution: backup, adopt, skip, or fail (default "fail")
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
      --repo string      dotfiles repository (default: found via $REPOS_DIR)
```

### SEE ALSO

* [dotty dotfiles](dotty_dotfiles.md)	 - Operate on the dotfiles repository dotty init generated.

