# Notes — ideas

Speculative / exploratory directions — not scoped, not committed, no design done yet.
Promote an idea to `enhancements.md` once it's actually going to be built next.

## Obsidian vault sync

The big deferred one from the original Phase 4 decision: sync with the Obsidian vault
at `~/personal/notes` instead of (or alongside) standalone storage. Explicitly
deferred when Notes was built — a git-based vault + markdown files + folder structure
is a materially bigger integration than standalone CRUD, and standalone was built
first on purpose. Revisit only if actually wanted.

## Folders / tags

Notes is a deliberately flat list today — nothing in the (mockup) reference app
suggested a structure was needed, so none was built speculatively. Worth adding if the
flat list becomes unwieldy in practice.

## Full-text search

Related to the "no search" enhancement, but a step further — proper full-text search
(Postgres `tsvector`, or an external index) rather than simple title/content filtering.

## Note linking

Wiki-style `[[note title]]` links between notes, with backlinks — meaningfully bigger
scope, closer to what a real Obsidian-sync would bring anyway.

## Pin / favorite

A lightweight `is_pinned` flag to keep a few notes at the top of the list, short of
full folders/tags.

## Image attachments

Notes currently store `content` as plain text only. Embedding images would need file
storage, which doesn't exist for any domain in life-base yet.
