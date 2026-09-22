# Bikes — ideas

Speculative / exploratory directions — not scoped, not committed, no design done yet.
Promote an idea to `enhancements.md` once it's actually going to be built next.

## Due-item notifications

Nothing in the app pushes/emails when a component or maintenance task becomes due —
you have to open the app and look. No notification system exists anywhere in
life-base yet (not Finance's Reminders either), so this would be a cross-cutting
addition, not Bikes-specific, if ever built.

## Cost rollup per bike

Total spent on a bike — sum of `service_logs.cost` + linked `gear.cost` +
proportional `supplies` cost — as a dashboard figure. Nothing currently aggregates
spend; each cost lives on its own row.

## GPX/route storage per activity

Activities currently store distance/duration/elevation as numbers, no route data.
Storing an uploaded or Strava-fetched GPX file (or just a polyline) would enable a
map view — meaningfully bigger scope than the current numeric-only shape.

## Component/gear catalog with typical lifespan presets

"Chain, typically replace every 3000km" style presets when adding a component, instead
of typing `replacement_interval_km` from memory each time.

## Shared gear across multiple bikes

`gear.bike_id` is a single optional link today (one bike or none). A shoe or helmet
realistically gets worn across all your bikes, not just one — a many-to-many
gear-to-bikes relationship would track accumulated distance more accurately for gear
that isn't bike-specific.

## Auto-backfill multiple pages of Strava history

`FetchActivities` takes `page`/`per_page` but the frontend has to drive pagination
itself. A "bulk import last N months" flow that pages through automatically would make
initial Strava backfill less tedious than confirming one page at a time.

## Weather-aware ride logging

Pull historical weather for an activity's date/location (temperature, conditions) —
purely a nice-to-have for ride context, not core to maintenance tracking.
