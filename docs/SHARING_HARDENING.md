# Sharing hardening (HUB_PLAN_V2 Phase D5)

Status of the D5 items after the vault-sync work. Nothing here has been
compiled or run (no Go toolchain in the authoring sandbox).

## Done

- **Sensitive guard**: `POST /v1/share-links` for a note with `sensitive = true`
  requires `confirm_sensitive: true` (HTTP 409, `code: sensitive_confirm_required`
  otherwise), rejects "never" and caps `expires_at` at 7 days. Deleted
  (`deleted_at`) notes cannot be shared. Past `expires_at` values are rejected
  for every resource type.
- **Audit log**: link creation writes an `audit_logs` row (`method = SHARE_CREATE`,
  path contains note id, sensitive flag, expiry). For sensitive notes it fails
  closed: if the audit insert fails the new link is revoked and the request
  errors. Revocation is `DELETE /v1/share-links/:id`, already logged by the
  existing `AuditLog()` middleware.
- **Org rendering for shares**: the public view returns `rendered_html` (built in
  "share mode": no note-to-note links, no image embeds, videos as plain links)
  and never the raw org source, `vault_path`, `org_id` or sensitivity fields.
  Property drawers are always hidden.
- **Headers**: `GET /public/s/:token` sends `Cache-Control: no-store` and
  `X-Robots-Tag: noindex, nofollow`.
- Frontend share modal: confirm checkbox + warning for sensitive notes, expiry
  presets (1h / 1d / 7d / 30d / never; never and >7d disabled when sensitive).

## Still open (need design decisions - not attempted)

1. **Public resolver runs inside the main API with the service-role key**
   (`ShareLinkRepository.FindByToken`, migration 038). Any bug in a public route
   has full DB reach. Options: a snapshot table written at link creation and
   served through a read-only DB role, or a separate minimal read-only process
   behind the tunnel. Note that snapshots also change semantics (edits after
   sharing are not reflected) - decide whether that is wanted.
2. **Token hash at rest**: `share_links.token` is stored raw and looked up by
   equality. Store `sha256(token)` instead and show the token once at
   creation. Needs a migration (public `053_*`/`054_*` numbering was reserved
   but not used) and a transition for existing links.
3. **Scoped image URLs for shared notes**: shared org notes currently show
   image embeds as plain text. Serving them needs expiring, token-scoped signed
   URLs (never the private bucket) and the on-demand placeholder in the shared
   view; blocked on attachment storage (index step copying `attachments/` to a
   private bucket) which is not built.
4. **Rate limit on the public route**: `/public/s/:token` is unauthenticated and
   skips `RateLimit()`. Add a per-IP limiter (requires trusting the platform
   proxy header, currently disabled via `SetTrustedProxies(nil)`).
5. **Sensitive-after-share**: a link created before a note was flagged sensitive
   keeps working until it expires/is revoked. A `confirmed_sensitive` column on
   `share_links` would allow resolving to 404 for links that were not
   explicitly confirmed.
6. **Subtree/collection shares** and **"links to unshared notes become plain
   text"** beyond the current rule (all `id:` links are plain text in shares).
7. Replace the hand-written org renderer with `go-org` + `bluemonday` once
   modules can be fetched; the renderer's escape-everything design is the
   safety baseline either way.
