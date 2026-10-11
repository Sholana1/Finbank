-- +goose Up
CREATE TABLE sessions (
    id            uuid        PRIMARY KEY,
    user_id       uuid        NOT NULL REFERENCES users (id),
    device_id     text        NOT NULL,
    ip_address    text        NOT NULL,
    user_agent    text        NOT NULL,
    created_at    timestamptz NOT NULL,
    expires_at    timestamptz NOT NULL,
    revoked_at    timestamptz,
    revoke_reason text,

    CONSTRAINT sessions_revoke_reason_check
        CHECK (revoke_reason IS NULL OR revoke_reason IN ('LOGOUT', 'REFRESH_TOKEN_REUSE')),
    CONSTRAINT sessions_revoked_together_check
        CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))
);

CREATE INDEX sessions_user_id_active_idx ON sessions (user_id) WHERE revoked_at IS NULL;

CREATE TABLE refresh_tokens (
    id         uuid        PRIMARY KEY,
    session_id uuid        NOT NULL REFERENCES sessions (id),
    token_hash bytea       NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,

    CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash)
);

CREATE INDEX refresh_tokens_session_id_idx ON refresh_tokens (session_id);

CREATE TABLE login_attempts (
    id         uuid        PRIMARY KEY,
    user_id    uuid        REFERENCES users (id),
    outcome    text        NOT NULL,
    device_id  text        NOT NULL,
    ip_address text        NOT NULL,
    user_agent text        NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT login_attempts_outcome_check
        CHECK (outcome IN ('SUCCEEDED', 'INVALID_CREDENTIALS', 'ACCOUNT_LOCKED', 'ACCOUNT_DISABLED'))
);

CREATE INDEX login_attempts_user_id_created_at_idx ON login_attempts (user_id, created_at DESC);

GRANT UPDATE (revoked_at, revoke_reason) ON sessions TO identity_app;
GRANT UPDATE (used_at) ON refresh_tokens TO identity_app;

-- +goose Down
DROP TABLE login_attempts;
DROP TABLE refresh_tokens;
DROP TABLE sessions;
