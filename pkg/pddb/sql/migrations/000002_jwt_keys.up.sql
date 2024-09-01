CREATE TABLE jwt_key_algorithms (
    jwt_key_algorithm_id SERIAL PRIMARY KEY,
    algorithm_name TEXT NOT NULL UNIQUE
);

CREATE TABLE jwt_keys (
    jwt_key_id SERIAL PRIMARY KEY,
    jwt_key_algorithm_id INTEGER NOT NULL REFERENCES jwt_key_algorithms(jwt_key_algorithm_id) ON DELETE RESTRICT,
    jwt_key_uuid UUID NOT NULL UNIQUE,
    key_secret_material BYTEA NOT NULL,
    generation_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expiration_timestamp TIMESTAMP NOT NULL
);

CREATE INDEX jwt_keys_algorithm_id_idx ON jwt_keys(jwt_key_algorithm_id);
CREATE INDEX jwt_keys_expiration_timestamp_idx ON jwt_keys(expiration_timestamp);