## dotty profile new

Create a system-level profile.

### Synopsis

Create a profile directory under $XDG_CONFIG_HOME/dotty/<name>. Without
--name dotty prompts for one. Unless --activate is given, dotty asks whether
to activate the new profile right away.

```
dotty profile new [flags]
```

### Examples

```
  dotty profile new
  dotty profile new --name=work --description="work laptop" --activate
```

### Options

```
      --activate             activate the profile after creating it
      --description string   short description of the profile
  -h, --help                 help for new
      --name string          name for the new profile
```

### Options inherited from parent commands

```
      --profile string   profile to operate on (defaults to the active profile)
```

### SEE ALSO

* [dotty profile](dotty_profile.md)	 - Manage system profiles that travel across machines.

