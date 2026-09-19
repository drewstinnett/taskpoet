# Recurring tasks

A recurring task is a template that spawns pending tasks as they come due. This
works like TaskWarrior, so tasks you import keep going where they left off.

```console
$ taskpoet add --recur weekly --due monday Water the plants
$ taskpoet add --recur 3months --chained --due 1w Change the furnace filter
```

The template needs a `--due` date to count from. It won't show up in `taskpoet
list`, its pending instances do. Use `--until` to end the series.

## Rules

`--recur` takes:

* names: `daily`, `weekdays`, `weekly`, `biweekly` (or `fortnight`), `monthly`,
  `bimonthly`, `quarterly`, `semiannual`, `annual` (or `yearly`), `biannual`
  (or `biyearly`)
* a count and a unit: `3d`, `2w`, `6months`, `1q`, `2y`, `12h`, `30min`

## Periodic (the default)

Every instance has a fixed due date: the template's due date plus a whole number
of periods, so it never drifts. Monthly from the 31st gives Jan 31, Feb 29,
Mar 31, Apr 30, and so on. Times stay the same on the wall clock across
daylight saving changes.

## Chained

With `--chained` there is only ever one pending instance. When you finish it,
the next one is due a period *after you finished*, which is right for things
like "every 3 months after the last time". If you delete an instance the chain
ends.

## When do they appear?

Whenever you run `taskpoet list`, and with `taskpoet recur`. It doesn't matter
how often, it only creates what is missing. `taskpoet recur --dry-run` shows
what it would create.

| Setting | Default | |
|---|---|---|
| `recurrence.enabled` | `true` | create instances automatically on `list` |
| `recurrence.limit` | `1` | how many future instances to have ready |
| `recurrence.catchup` | `latest` | what to do about periods you missed |

**Catching up.** If nothing ran for a month, a daily task missed 30 periods.
`latest` creates just the most recent one, so you aren't buried in stale tasks.
`all` creates every missed one, like TaskWarrior does (at most 1000 per template
in one go). A series whose `until` date has passed doesn't create anything on
`latest`.

Deleting a template also deletes its pending instances. Finished ones stay as
history.
