-- Migration 036: page_access for the Caption Library's own sub-page, now
-- split out of the Social page into /social/captions (own nav entry, same
-- pattern as Finances' sub-pages).

insert into page_access (role, page_key, allowed)
values ('admin', 'social-captions', true), ('member', 'social-captions', true)
on conflict (role, page_key) do nothing;
