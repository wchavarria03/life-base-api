-- Migration 022: caption template library for social posts.
--
-- Every save (create, "customize", "create new version") inserts a new
-- caption_template_versions row; caption_templates.current_version_id just
-- points at whichever one is active, so full history is automatic rather
-- than a special case. Categories are a managed per-user list, assigned to
-- templates and to individual posts via many-to-many junction tables.
--
-- Grants are included here up front (migration 019 shipped without them
-- and broke /v1/me in prod with "42501: permission denied" — see
-- migration 021's fix).

create table caption_categories (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    name       text not null,
    created_at timestamptz not null default now(),
    unique (user_id, name)
);

create table caption_templates (
    id                 uuid primary key default gen_random_uuid(),
    user_id            uuid not null references auth.users(id) on delete cascade,
    title              text not null,
    current_version_id uuid,
    created_at         timestamptz not null default now(),
    updated_at         timestamptz not null default now()
);

create table caption_template_versions (
    id              uuid primary key default gen_random_uuid(),
    template_id     uuid not null references caption_templates(id) on delete cascade,
    body            text not null,
    version_number  int not null,
    created_at      timestamptz not null default now(),
    unique (template_id, version_number)
);

alter table caption_templates
    add constraint caption_templates_current_version_fk
    foreign key (current_version_id) references caption_template_versions(id);

create table caption_template_categories (
    template_id uuid not null references caption_templates(id) on delete cascade,
    category_id uuid not null references caption_categories(id) on delete cascade,
    primary key (template_id, category_id)
);

create table social_post_categories (
    social_post_id uuid not null references social_posts(id) on delete cascade,
    category_id    uuid not null references caption_categories(id) on delete cascade,
    primary key (social_post_id, category_id)
);

create index on caption_templates (user_id);
create index on caption_template_versions (template_id);
create index on caption_template_categories (category_id);
create index on social_post_categories (category_id);

-- ── RLS ──────────────────────────────────────────────────────────────────────

alter table caption_categories          enable row level security;
alter table caption_templates           enable row level security;
alter table caption_template_versions   enable row level security;
alter table caption_template_categories enable row level security;
alter table social_post_categories      enable row level security;

create policy "caption_categories: user owns rows"
on caption_categories for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

create policy "caption_templates: user owns rows"
on caption_templates for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

create policy "caption_template_versions: user owns rows via template"
on caption_template_versions for all
using (
    template_id in (select id from caption_templates where user_id = auth.uid())
)
with check (
    template_id in (select id from caption_templates where user_id = auth.uid())
);

create policy "caption_template_categories: user owns rows via template"
on caption_template_categories for all
using (
    template_id in (select id from caption_templates where user_id = auth.uid())
)
with check (
    template_id in (select id from caption_templates where user_id = auth.uid())
);

create policy "social_post_categories: user owns rows via post"
on social_post_categories for all
using (
    social_post_id in (select id from social_posts where user_id = auth.uid())
)
with check (
    social_post_id in (select id from social_posts where user_id = auth.uid())
);

-- ── Grants ───────────────────────────────────────────────────────────────────

grant all on table caption_categories          to anon, authenticated, service_role;
grant all on table caption_templates            to anon, authenticated, service_role;
grant all on table caption_template_versions    to anon, authenticated, service_role;
grant all on table caption_template_categories  to anon, authenticated, service_role;
grant all on table social_post_categories       to anon, authenticated, service_role;
