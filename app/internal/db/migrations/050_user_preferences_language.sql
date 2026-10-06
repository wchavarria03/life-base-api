alter table user_preferences
    add column language text not null default 'en' check (language in ('en', 'es'));
