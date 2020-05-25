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

CREATE ROLE shared_anon nologin;

GRANT USAGE ON SCHEMA monitor TO shared_anon;
GRANT SELECT ON monitor.users TO shared_anon;

CREATE ROLE authenticator noinherit login password 'mysecretpassword';
GRANT shared_anon TO authenticator;

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

REVOKE shared_anon FROM authenticator;
DROP ROLE authenticator;

REVOKE SELECT ON monitor.users FROM shared_anon;
REVOKE USAGE ON SCHEMA monitor FROM shared_anon;
DROP ROLE shared_anon;

DROP INDEX monitor.users_index_token;
DROP TABLE monitor.users;
DROP SCHEMA monitor;