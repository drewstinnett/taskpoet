# Configuration

## Config file

Settings live in `~/.taskpoet.yaml` (or wherever `--config` points):

```yaml
dbpath: /path/to/taskpoet.db  # see below for the default
theme: solarized-dark         # default, solarized-light or solarized-dark
defaults:
  due: 2d                     # due date for new tasks that don't say
recurrence:
  enabled: true
  limit: 1
  catchup: latest             # or all
update:
  check: true                 # look for new releases now and then, see below
```

Every setting can also come from the environment, with a `TASKPOET_` prefix and
underscores for the dots: `TASKPOET_DBPATH`, `TASKPOET_THEME`,
`TASKPOET_DEFAULTS_DUE`, `TASKPOET_RECURRENCE_LIMIT`. Flags win over the
environment, and the environment wins over the file.

## Update checks

Now and then taskpoet looks for a newer release, and says so after the command
you ran, on stderr:

```plain
A new version of taskpoet is available: v2.1.0 (this is v2.0.0)
Update it with: taskpoet update
```

It asks GitHub once every six hours at most, in the background, and holds up a
command for a second at the very most while waiting for the answer. It says so
once every six hours as well, not on every command. It is quiet when it isn't
run from a terminal (scripts, cron, pipes), when `CI` is set, for shell
completion, and for development builds. What it remembers is in
`taskpoet/update-check.json` in your user cache directory, and can be deleted.

To turn it off, set `update.check: false`, or `TASKPOET_UPDATE_CHECK=false`.
`taskpoet update` still works when it is off.

## The database

Everything is stored in a single [bbolt](https://github.com/etcd-io/bbolt) file.
It defaults to `$XDG_DATA_HOME/taskpoet/taskpoet.db`, or
`~/.local/share/taskpoet/taskpoet.db` when that isn't set. Change it with the
`--db` flag or the `dbpath` setting.

Only one taskpoet can have the file open at a time. If another one does, you get
an error after a second rather than a hang.

### Namespaces

`--namespace` (`-n`) keeps separate sets of tasks in the same file, for instance
`taskpoet -n work list`. The default is `default`.

### Layout

Inside a namespace, tasks are keyed by UUID, and the status, recurrence parent
and dependencies are indexed, so listing and looking up by the start of an id
don't scan everything. A `schema_version` is stored too. taskpoet refuses to
open a file written by a newer version, or by v0.x (see [Upgrading](upgrading.md)).

## Task ids

Tasks are identified by their UUID, and commands take any unique start of it,
like `taskpoet done 1b35`. `taskpoet list` shows the first 8 characters. If a
start matches more than one task you are told which ones.
