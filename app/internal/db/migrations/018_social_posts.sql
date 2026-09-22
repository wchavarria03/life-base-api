-- Migration 018: social posts (Facebook + Instagram)

create table social_posts (
    id                  uuid primary key default gen_random_uuid(),
    user_id             uuid not null references auth.users(id) on delete cascade,
    filename            text not null,
    caption             text not null,
    facebook_status     text not null check (facebook_status in ('success','failed','skipped')),
    facebook_post_id    text,
    facebook_photo_url  text,
    facebook_permalink  text,
    facebook_error      text,
    instagram_status    text not null check (instagram_status in ('success','failed','skipped')),
    instagram_media_id  text,
    instagram_permalink text,
    instagram_error     text,
    created_at          timestamptz not null default now()
);

create index on social_posts (user_id);
create index on social_posts (created_at);

alter table social_posts enable row level security;

create policy "social_posts: user owns rows"
on social_posts for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table social_posts to anon, authenticated, service_role;
