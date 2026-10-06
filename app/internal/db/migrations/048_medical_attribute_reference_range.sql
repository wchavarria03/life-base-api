-- Migration 048: reference range on medical attribute defs — the
-- expected-normal range shown on the lab report (e.g. "4.80 - 10.80
-- 10^3/uL"), displayed alongside the label/description.

alter table medical_attribute_defs add column reference_range text;
