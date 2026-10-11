-- +goose Up
CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    email         text        NOT NULL,
    phone_number  text        NOT NULL,
    password_hash text        NOT NULL,
    status        text        NOT NULL,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL,

    CONSTRAINT users_email_key        UNIQUE (email),
    CONSTRAINT users_phone_number_key UNIQUE (phone_number),
    CONSTRAINT users_status_check     CHECK (status IN ('PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'CLOSED'))
);

-- +goose Down
DROP TABLE users;
