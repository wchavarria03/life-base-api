-- Notes domain — pinned flag, so a frequently-used note (e.g. a grocery
-- list) doesn't get buried under a long version history of nutrition-plan
-- style notes. Pinned notes sort first in the list.

alter table notes.notes add column pinned boolean not null default false;
