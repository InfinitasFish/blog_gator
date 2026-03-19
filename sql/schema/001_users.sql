-- +goose Up
CREATE TABLE users (
    id int PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT UNIQUE
);

-- +goose Down
DROP TABLE users;