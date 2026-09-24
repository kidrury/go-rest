CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    refresh_token_hash BYTEA NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    last_used_at TIMESTAMPTZ,

    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES sessions(id)
);

CREATE INDEX sessions_user_id_idx
    ON sessions(user_id);

CREATE INDEX sessions_family_id_idx
    ON sessions(family_id);

CREATE INDEX sessions_expires_at_idx
    ON sessions(expires_at);