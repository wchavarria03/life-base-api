# Tasks (Household/House/TODO) — enhancements

Concrete, scoped improvements to the existing generic tasks domain — things we know how
to build, just haven't yet. Speculative/exploratory stuff belongs in `ideas.md` instead.

## `complete` always uses today's date, can't log a backdated completion

`POST /v1/tasks/:id/complete` sets `last_completed_date`/`completed_at` to *now* — no
way to say "I actually did this yesterday" after the fact. For a recurring task
especially, this skews the next due date by however late you were logging it. Accept
an optional `completed_date` in the request body, default to today when omitted.

## No completion history for recurring tasks

Unlike Bikes' `component_history` (which logs every replacement), a recurring task's
past completions aren't recorded anywhere — `complete` only overwrites
`last_completed_date` in place. No way to see "cleaned the bathroom 8 times this
quarter" or build a streak view. Would need a small `task_completions` log table,
written by `complete` in addition to updating the task row.

## Recurring due-ness is binary — no "due soon" state

`status` for a recurring task is only `overdue` or `upcoming` — there's no equivalent
of one-off tasks' `due_today`. A task due in 2 days looks identical to one due in 3
weeks. Worth adding a "due soon" window (e.g. within X% of `interval_days`) so the
frontend can surface it before it's actually overdue.

## No snooze

The original Household design note ("frequency/interval, last completed, **snooze**")
called for a snooze action — push a task's next due date out without marking it
complete. Not built. Would need either a `snoozed_until` field or a
`last_completed_date` write that doesn't imply "was actually done."
