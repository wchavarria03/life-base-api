# Attachments (document_links)

Migration `055_document_links.sql` adds `documents.sensitive` and `document_links`
(`document_id`, `entity_type`, `entity_id`; types: `note`, `task_log`, `bike`, `component`, `medical_record`).

Routes (`/v1`, JWT): `POST/GET /documents/:id/links`, `DELETE /documents/:id/links/:linkId`,
`GET /documents/by-entity?type=&id=`, `GET /documents/:id/signed-url`; upload accepts `sensitive=true`.

Signed URLs: `GET /public/documents/:id?exp=&sig=` (HMAC-SHA256 of `id|exp`, key `DOCUMENT_URL_SECRET`;
unset = signed URLs and share-link attachments are disabled). TTL 1 h, sensitive documents 5 min. Only raster
images and PDFs are served inline.

Share links list only non-sensitive documents linked to the shared note. Links are owner-scoped by RLS; the
entity id itself is not verified to exist.
