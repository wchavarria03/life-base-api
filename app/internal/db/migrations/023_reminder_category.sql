-- Migration 023: category on reminders, for "recurring transactions" —
-- extending the existing Reminders feature (recurring rule + due occurrence
-- + manual confirm) rather than building a separate concept. category_id
-- lets Complete/Link auto-categorize the real transaction the same way a
-- category is applied anywhere else.

alter table payment_reminders
    add column category_id uuid references categories(id);
