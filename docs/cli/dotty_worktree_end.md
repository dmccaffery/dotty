## dotty worktree end

Remove a worktree, its agent/* branch, and its tmux session.

### Synopsis

Tear a worktree down: kill the tmux session named after it, remove the
worktree from its repository, and delete its branch — but only an agent/*
branch, so a manually checked-out branch survives. Uncommitted or unpushed
work is reported first; a worktree that is already gone exits quietly.

path falls back to the WorktreeRemove hook JSON on stdin
({"worktree_path": ...}).

```
dotty worktree end [path] [flags]
```

### Examples

```
  dotty worktree end ~/Repos/dotty/.worktrees/dotty-fix-1
  echo '{"worktree_path":"..."}' | dotty worktree end   # hook form
```

### Options

```
  -h, --help   help for end
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty worktree](dotty_worktree.md)	 - Manage agent git worktrees.

