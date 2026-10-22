-- +migrate Up

CREATE TABLE users (
    id UUID PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    translator TEXT NOT NULL,
    status TEXT NOT NULL,
    result TEXT NOT NULL DEFAULT ''
);

-- +migrate Down

DROP TABLE tasks;
DROP TABLE users;