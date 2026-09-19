# Coming from TaskWarrior

TaskPoet can import your whole TaskWarrior database from `task export`, and
keeps everything TaskWarrior knows about a task.

## Importing

### From an export file

If you already have an export, or Taskwarrior lives on a different machine,
you don't need Taskwarrior installed here at all. On the machine that has it:

```console
$ task export > tw.json
```

Then, wherever TaskPoet is:

```console
$ taskpoet import tw.json
```

Use `-` to read from stdin instead, so it works in a pipe:

```console
$ ssh other-machine task export | taskpoet import -
```

The file can be a JSON array (what `task export` prints), one JSON object per
line, or several of those one after the other. UTF-8 with or without a byte
order mark and UTF-16 are all fine, so an export saved by Windows PowerShell
(`task export > tw.json`) works too.

`taskpoet import taskwarrior tw.json` is the same thing, spelled out.

### Straight from Taskwarrior

If Taskwarrior is installed here, let TaskPoet run it for you:

```console
$ taskpoet import --from-task
```

This runs `task export` once for each of `pending`, `waiting`, `completed`,
`deleted` and `recurring`, so nothing is left out no matter which version of
TaskWarrior you have. It honors your `TASKRC` and `TASKDATA` settings, and
`--task-bin` points it at a different `task` binary.

A plain `task export` should include everything too. If the counts you get
(see below) come up short of what `task count` says, use `--from-task`, which
asks for each status by name.

### Check before you commit

```console
$ taskpoet import --from-task --dry-run
Dry run, no tasks were written.
Read 1873 tasks from Taskwarrior [completed 1410 deleted 212 pending 236 recurring 15]
Would import 1873 new tasks, 0 overwritten, 0 already present and skipped
```

Compare those numbers with `task count status:pending` (and `completed`,
`deleted`, `recurring`). Then run it again without `--dry-run`. This works the
same with an export file: `taskpoet import tw.json --dry-run`.

### Importing again

Tasks are matched by their UUID, which TaskPoet keeps, so importing twice is
safe. Tasks that are already there are left alone. Use `--overwrite` to replace
them with what TaskWarrior has now (taskpoet's own effort/impact on a task is
kept). The whole import is a single transaction: if anything goes wrong, nothing
is written.

A record that can't be converted (for instance one with an unknown status) stops
the import and says which one it was. `--skip-invalid` skips those records and
carries on, with a warning for each.

## What comes across

| TaskWarrior | TaskPoet |
|---|---|
| `uuid` | the task id, kept as is. `taskpoet describe` takes any unique start of it |
| `status` pending, completed, deleted, recurring | the same |
| `status` waiting (older versions) | pending, with the `wait` time kept |
| `description`, `project`, `priority` (H/M/L), `tags` | the same |
| `entry`, `modified`, `start`, `end`, `due`, `wait`, `until`, `scheduled`, `reviewed` | the same, `wait` hides the task until then |
| `depends` | the same, blocked tasks are marked and rank lower |
| `annotations` | the same, `taskpoet annotate` adds more |
| `recur`, `rtype`, `mask`, `parent`, `imask` | the same, see [Recurring tasks](recurring.md) |
| any user defined attribute | kept as is, and shown by `taskpoet describe` |
| `id` (the 1, 2, 3 numbers) | not kept, they change all the time. Use the start of the UUID |
| `urgency` | not kept, taskpoet works its own out |

A few things that are worth knowing:

* Older TaskWarrior versions wrote `tags` and `depends` as a comma separated
  string, and annotations as `annotation_<epoch>` keys. Those are understood.
* If you changed which values `priority` can have, tasks with something other
  than H, M or L keep it as a UDA named `priority` instead of losing it.
* A task without an `entry` time gets its `modified` time, or the time of the
  import.
* References to tasks that aren't in the database, dependency cycles, and
  recurring templates that have no rule are reported as warnings. They are
  kept as they are, TaskWarrior allows them too.

## Getting a copy back out

`taskpoet export` writes the same JSON format to stdout, which is handy for
backups. taskpoet's own `effort_impact` is included as an extra field.
