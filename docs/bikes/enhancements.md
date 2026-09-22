# Bikes — enhancements

Concrete, scoped improvements to the existing Bikes/gear maintenance feature — things
we know how to build, just haven't yet. Speculative/exploratory stuff belongs in
`ideas.md` instead.

## Deleting an activity doesn't reverse its accumulated wear

`ActivityService.Delete` removes the row but never undoes what `Create` did — the
bike's `mileage` and every active component's/gear item's `accumulated_km` stay
inflated by that activity's `distance_km` forever. Correcting a bad activity entry
(wrong distance, duplicate import) currently requires manually adjusting the bike and
every affected component/gear afterward. Delete should subtract the same amounts it
added.

## No duplicate guard on Strava-imported activities

`GET /v1/bikes/strava/activities` returns a preview; nothing stops the same Strava ride
from being confirmed and `POST`ed twice, double-counting its distance onto mileage and
wear. Check `strava_activity_id` for an existing match on that bike before creating.

## No aggregate "what needs attention" endpoint across bikes

Component/maintenance due-ness is only queryable per-bike (`GET
/v1/bikes/:id/components`, `.../maintenance-tasks`). A dashboard summary ("3 items due
across all bikes") currently means fetching every bike then fanning out N more requests
per bike. Worth a `GET /v1/bikes/due` (or similar) that aggregates across all of a
user's bikes in one call if the Hub dashboard's per-bike fan-out ever becomes a real
perf concern.

## `components.accumulated_km` is directly PATCH-able, bypassing the replacement log

`PATCH /v1/bikes/:id/components/:componentId` accepts a raw partial merge, including
`accumulated_km` — so it can be reset without going through `POST .../replace`, which
is supposed to be the only path that resets wear (and logs why, via
`component_history`). Either block `accumulated_km` from the raw-merge fields, or accept
that `replace` is a convention, not an enforced one.

## No activity edit — only delete-and-recreate

Documented as intentional ("Activities aren't edited after the fact — correct by
deleting and re-creating"), but combined with the wear-reversal gap above, that
workaround doesn't actually work cleanly yet. Once delete correctly reverses
accumulation, delete-and-recreate becomes a real option; until then, an `Update`
endpoint might be less error-prone for simple corrections (fixing a typo in the name,
say) that don't need to touch the wear math at all.
