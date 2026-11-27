-- uploads: the pictures the shop's team sends (docs/architecture/erd.md §3.20).
--
-- The backend container is read-only with no volume, so a picture is a row. The
-- name is the server's own (32 random hex characters and an extension chosen from
-- the detected type): a client-supplied name or path can never be stored. data is
-- selected only by GET /v1/uploads/{name}, never by a list.
CREATE TABLE uploads (
    id           uuid        NOT NULL DEFAULT gen_random_uuid(),
    name         text        NOT NULL,
    content_type text        NOT NULL,
    size_bytes   integer     NOT NULL,
    data         bytea       NOT NULL,
    uploaded_by  uuid        NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uploads_pkey PRIMARY KEY (id),
    CONSTRAINT uq_uploads_name UNIQUE (name),
    CONSTRAINT ck_uploads_name_format CHECK (name ~ '^[a-f0-9]{32}\.(jpg|png|webp)$'),
    CONSTRAINT ck_uploads_content_type CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    CONSTRAINT ck_uploads_size_range CHECK (size_bytes BETWEEN 1 AND 5242880),
    CONSTRAINT fk_uploads_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES users (id) ON DELETE SET NULL
);

-- A picture is already compressed: TOAST need not try again.
ALTER TABLE uploads ALTER COLUMN data SET STORAGE EXTERNAL;
