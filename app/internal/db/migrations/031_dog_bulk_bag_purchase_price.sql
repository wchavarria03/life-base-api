-- Migration 031: track purchase date + price on bulk bags instead of frozen
-- date. No production data yet, so we rename the column in place rather than
-- carrying a parallel one.

alter table dog_bulk_bags rename column frozen_date to purchase_date;
alter table dog_bulk_bags add column price numeric;
