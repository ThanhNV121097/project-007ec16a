-- Accounts and sessions (plan item 9, "Dealer sign-in and protected admin area").
-- users holds all three kinds of account (admin, staff, customer); sessions holds
-- the SHA-256 of each opaque session token. Both tables are new, so this file
-- locks nothing that exists. The admin row is not made here: the server makes it
-- at start-up from the owner's settings.

CREATE TABLE users (
    id            uuid        NOT NULL DEFAULT gen_random_uuid(),
    email         text        NOT NULL,
    name          text        NOT NULL DEFAULT '',
    phone         text        NOT NULL DEFAULT '',
    address       text        NOT NULL DEFAULT '',
    city          text        NOT NULL DEFAULT '',
    password_hash text        NOT NULL,
    role          text        NOT NULL,
    is_active     boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT ck_users_email_lowercase CHECK (email = lower(email) AND email <> '' AND length(email) <= 200),
    CONSTRAINT ck_users_email_ascii CHECK (email ~ '^[!-~]+$'),
    CONSTRAINT ck_users_password_hash_not_blank CHECK (password_hash <> ''),
    CONSTRAINT ck_users_role CHECK (role IN ('admin', 'staff', 'customer'))
);

-- At most one admin row, whatever writes the table.
CREATE UNIQUE INDEX uq_users_one_admin ON users ((true)) WHERE role = 'admin';

CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE sessions (
    id         uuid        NOT NULL DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL,
    token_hash bytea       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    CONSTRAINT pk_sessions PRIMARY KEY (id),
    CONSTRAINT uq_sessions_token_hash UNIQUE (token_hash),
    CONSTRAINT ck_sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT ck_sessions_expiry_after_created CHECK (expires_at > created_at),
    CONSTRAINT fk_sessions_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE NO ACTION
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);
