CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    user_uuid UUID NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    user_creation_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE password_credentials (
    password_credential_id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE UNIQUE,
    hashing_method BYTEA NOT NULL,
    password_hash BYTEA NOT NULL,
    password_salt BYTEA NOT NULL,
    password_hashing_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);