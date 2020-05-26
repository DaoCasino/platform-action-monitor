-- migrate:up
CREATE SCHEMA monitor;

CREATE TABLE monitor.users
(
    id            serial primary key,
    token         character varying(64) not null,
    description   text,
    creation_date timestamp without time zone
);

CREATE INDEX users_index_token ON monitor.users USING btree (token, id);

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION monitor.random_token() RETURNS character varying(64) AS $$
    SELECT encode(gen_random_bytes(32), 'hex')
$$ LANGUAGE SQL VOLATILE;

CREATE OR REPLACE FUNCTION monitor.create_user() RETURNS character varying(64) AS $$
    INSERT INTO monitor.users (token, creation_date) VALUES (monitor.random_token(), now()) returning token;
$$ LANGUAGE SQL VOLATILE;

-- migrate:down
DROP FUNCTION monitor.create_user();
DROP FUNCTION monitor.random_token();

DROP INDEX monitor.users_index_token;
DROP TABLE monitor.users;
DROP SCHEMA monitor;