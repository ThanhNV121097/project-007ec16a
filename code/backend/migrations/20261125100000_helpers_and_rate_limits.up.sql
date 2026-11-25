-- Helpers shared by every later story, and the rate-limit counters.
-- Created here once so the search, track and sign-in stories, which run in
-- parallel after this one, never create them twice.

-- doi_fold: lower case with every accent removed, so "Cửa Lò" is found by
-- "cua lo". The text is decomposed (NFD) and the combining marks U+0300 to
-- U+036F are dropped, which covers every Vietnamese tone and vowel mark in both
-- cases; the one letter that does not decompose, đ/Đ, is replaced by hand.
-- Characters outside Vietnamese fold the same way, and nothing depends on the
-- database's locale. NULL in, NULL out. Needs a UTF-8 database.
CREATE OR REPLACE FUNCTION doi_fold(t text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$
    SELECT lower(replace(replace(
        regexp_replace(normalize(t, NFD), '[' || chr(768) || '-' || chr(879) || ']', '', 'g'),
        'đ', 'd'), 'Đ', 'D'))
$$;

DO $$
BEGIN
    ASSERT doi_fold('Cửa Lò') = 'cua lo', 'doi_fold must fold Cửa Lò to cua lo';
    ASSERT doi_fold('Đại') = 'dai', 'doi_fold must fold Đại to dai';
    ASSERT doi_fold('100%') = '100%', 'doi_fold must leave 100% as it is';
    ASSERT doi_fold(NULL) IS NULL, 'doi_fold(NULL) must be NULL';
END $$;

-- set_updated_at: the trigger function that moves a record's updated_at, so no
-- handler can forget to.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END
$$;

-- rate_limits: one row per (scope, hashed subject), the failure counters of
-- sign-in, order lookup and sign-up. A hash, so the table holds no email, no
-- network address and no order number. No endpoint of this story touches it.
CREATE TABLE IF NOT EXISTS rate_limits (
    scope             text        NOT NULL,
    subject_hash      bytea       NOT NULL,
    hits              integer     NOT NULL DEFAULT 0,
    window_started_at timestamptz NOT NULL DEFAULT now(),
    locked_until      timestamptz,
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT rate_limits_pkey PRIMARY KEY (scope, subject_hash),
    CONSTRAINT ck_rate_limits_scope CHECK (scope IN ('sign_in_email', 'sign_in_address', 'order_lookup_number', 'order_lookup_address', 'sign_up_address')),
    CONSTRAINT ck_rate_limits_subject_hash_length CHECK (octet_length(subject_hash) = 32),
    CONSTRAINT ck_rate_limits_hits_not_negative CHECK (hits >= 0)
);

CREATE INDEX IF NOT EXISTS idx_rate_limits_window_started_at ON rate_limits (window_started_at);

DROP TRIGGER IF EXISTS trg_rate_limits_updated_at ON rate_limits;
CREATE TRIGGER trg_rate_limits_updated_at BEFORE UPDATE ON rate_limits
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
