DROP INDEX jwt_keys_algorithm_id_idx;
DROP INDEX jwt_keys_expiration_timestamp_idx;
DROP INDEX jwt_keys_jwt_key_uuid_idx;

DROP TABLE jwt_keys;
DROP TABLE jwt_key_algorithms;