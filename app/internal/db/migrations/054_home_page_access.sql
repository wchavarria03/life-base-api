-- Page access for the Home maintenance dashboard (/home).
insert into page_access (role, page_key, allowed)
values ('admin', 'home', true), ('member', 'home', true)
on conflict (role, page_key) do nothing;
