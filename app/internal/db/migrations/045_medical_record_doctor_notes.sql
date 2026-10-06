-- Migration 045: doctor name and recommendations on medical_records —
-- structured fields alongside the free-text notes, so "who saw you" and
-- "what they suggested" stay queryable/consistent across records instead of
-- buried in prose.

alter table medical_records add column doctor_name text;
alter table medical_records add column recommendations text;
