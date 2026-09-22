-- Migration 002: bikes.bikes gains serial_number and notes.
-- notes is freeform — same pattern as components/service_logs/tasks already use;
-- intended use includes pasting links to maintenance schedules/owner's manuals.

alter table bikes.bikes add column if not exists serial_number text;
alter table bikes.bikes add column if not exists notes text;
