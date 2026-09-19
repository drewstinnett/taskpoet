# Upgrading from v0.x

Version 2 is a deliberate break, so the storage and the task model could be
redone properly. There is no migration of a v0.x database. Bring your tasks over
from TaskWarrior instead, see [Coming from TaskWarrior](taskwarrior.md). If you
used `taskpoet import tw.json` before, that still works, and now brings in
everything.

## What changed

* **New database.** The file is at `$XDG_DATA_HOME/taskpoet/taskpoet.db` now,
  not `~/.taskpoet.db`, and has a new layout. If v2 is pointed at a v0.x file it
  stops and tells you, it never changes it. Keep the old file if you might want
  the old version back.
* **Go module.** It is `github.com/drewstinnett/taskpoet/v2`.
* **Task model.** Tasks now have Taskwarrior's fields: a status, project,
  priority, dependencies, annotations, recurrence and UDAs. Some names changed:
  `hide_until` is `wait`, `cancel_after` is `until`, comments are annotations.
* **Ids** are the start of the task UUID, 8 characters instead of 5.
* **Recurrence** is real recurrence now (`--recur`), the `RecurringTasks`
  setting is gone.
* **Commands.** `active`/`get` is `list` (the old names still work), `comment` is
  `annotate` (still works), `--parent` on `add` is `--depends`. New: `import`,
  `export`, `recur`, `delete`. The `c` alias now only means `done`.
* **Removed.** The HTTP server, the TUI and the task source plugins, along with
  the hidden `server`, `ui`, `plugins` and `debug` commands.
* **Environment.** Settings from the environment need a `TASKPOET_` prefix, so
  `DBPATH` is now `TASKPOET_DBPATH`.
