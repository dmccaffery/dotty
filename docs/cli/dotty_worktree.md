## dotty worktree

Manage agent git worktrees.

### Synopsis

The agent worktree lifecycle: start creates or reuses a worktree on an
agent/<name> branch and prints its path; end removes the worktree, its tmux
session, and the agent/* branch. Both verbs also accept their agent hook
JSON on stdin (WorktreeCreate and WorktreeRemove), so they wire directly
into claude's hooks.

Worktrees live at the location the active profile configures (dotty init
--worktrees): a directory inside each repository — the default .worktrees,
kept out of git by the shared ignore — or one absolute shared root. Inside
them, commit and tag signing is off (sandboxed agents cannot reach the
security key); re-sign afterwards with dotty git resign.

### Examples

```
  dotty worktree start myrepo fix-tests
  dotty worktree end
```

### Options

```
  -h, --help   help for worktree
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty](dotty.md)	 - Utilities for a terminal-driven workflow and dotfiles.
* [dotty worktree end](dotty_worktree_end.md)	 - Remove a worktree, its agent/* branch, and its tmux session.
* [dotty worktree start](dotty_worktree_start.md)	 - Create or reuse an agent/* worktree and print its path.

