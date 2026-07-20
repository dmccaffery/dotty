## dotty git propose

Open or update trunk-based PRs for the stack.

### Synopsis

Push stack branches and open pull requests against upstream/main
(or origin/main). Default: layers from the trunk through the current branch.
With --all, propose every layer in the stack.

A branch without stack lineage works too: propose adopts it first — as a
discovered chain when the local branch topology makes one obvious, otherwise
as a new single-layer stack. Before any PR opens, every proposed layer must
be up to date with trunk (fast-forwardable) and with the layers below it; if
the stack has diverged or a lower layer gained commits, you are prompted to
rebase + resign, as `dotty git sync` does.

Each PR body includes a stack map with links. For multi-commit layers you pick
which commit supplies the title and description.

With --auto-merge=<merge|rebase|squash>, each proposed PR is flagged to merge
automatically with that method once its requirements pass. If the repository
has auto-merge switched off, or disallows the chosen method, propose warns and
continues. --auto-merge=comment instead posts a `/auto-merge` comment
on each PR, for repositories where a merge bot watches for it.

With --browse, each proposed PR opens in your browser afterwards; with --copy,
the PR URLs (one per line) land on your clipboard. Make any of these the
default via git configuration: `git config set dotty.propose.browse true`
(and dotty.propose.copy, dotty.propose.auto-merge).

```
dotty git propose [--all] [--auto-merge=merge|rebase|squash|comment] [--browse] [--copy] [flags]
```

### Examples

```
  dotty git propose
  dotty git propose --all
  dotty git propose --auto-merge=rebase
  dotty git propose --auto-merge=comment
  dotty git propose --browse --copy
```

### Options

```
      --all               propose every layer in the stack, not only through the current branch
      --auto-merge mode   auto-merge each proposed pull request with the given method (merge, rebase, or squash); mode comment posts a /auto-merge comment for a merge bot instead
      --browse            open each proposed pull request in the browser
      --copy              copy the proposed pull request URL(s) to the clipboard
  -h, --help              help for propose
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty git](dotty_git.md)	 - Git helpers built on dotty's commit signing.

