-- Migration 027: scheduled social posts + per-network Instagram caption.
--
-- Supabase Storage bucket for images that need to persist until their
-- scheduled send time (today's immediate-post flow never stores images —
-- it streams straight to Meta and relies on Meta's own CDN URL for
-- Instagram). The bucket is public: Instagram's Graph API fetches the
-- image URL unauthenticated, so it must be reachable without our own
-- auth headers.

insert into storage.buckets (id, name, public)
values ('social-images', 'social-images', true)
on conflict (id) do nothing;

-- Uploads restricted to the owning user (object path is expected to be
-- "<user_id>/<filename>"); reads are public per the bucket's own setting,
-- so no select policy is needed here.
create policy "social-images: user uploads own folder"
on storage.objects for insert
with check (
    bucket_id = 'social-images'
    and (storage.foldername(name))[1] = auth.uid()::text
);

create policy "social-images: user manages own folder"
on storage.objects for all
using (
    bucket_id = 'social-images'
    and (storage.foldername(name))[1] = auth.uid()::text
)
with check (
    bucket_id = 'social-images'
    and (storage.foldername(name))[1] = auth.uid()::text
);

-- ── social_posts: optional distinct Instagram caption ───────────────────────
-- caption remains the primary/Facebook caption; caption_instagram is only
-- set when it differs, so existing rows and the existing frontend (which
-- just reads `caption`) keep working unchanged.

alter table social_posts
    add column caption_instagram text;

-- ── scheduled_posts ──────────────────────────────────────────────────────────

create table scheduled_posts (
    id                    uuid primary key default gen_random_uuid(),
    user_id               uuid not null references auth.users(id) on delete cascade,
    storage_path          text not null,
    filename              text not null,
    caption_facebook      text not null,
    caption_instagram     text,
    post_facebook         boolean not null default true,
    post_instagram        boolean not null default true,
    category_ids          uuid[] not null default '{}',
    scheduled_at          timestamptz not null,
    status                text not null default 'pending' check (status in ('pending','posted','failed','canceled')),
    result_social_post_id uuid references social_posts(id),
    error                 text,
    created_at            timestamptz not null default now(),
    updated_at            timestamptz not null default now()
);

create index on scheduled_posts (user_id);
create index on scheduled_posts (status, scheduled_at);

alter table scheduled_posts enable row level security;

create policy "scheduled_posts: user owns rows"
on scheduled_posts for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table scheduled_posts to anon, authenticated, service_role;
