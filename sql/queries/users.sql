-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, user_name)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE users.user_name = $1
LIMIT 1;

-- name: ResetTable :exec
DELETE FROM users;

-- name: ListUsers :many
SELECT users.user_name FROM users
ORDER BY users.id;

-- name: GetFeedFollowsForUser :many
SELECT users.user_name, feeds.feed_name, feeds.feed_url FROM users
INNER JOIN feed_follows ON users.id = feed_follows.user_id
INNER JOIN feeds ON feeds.id = feed_follows.feed_id
WHERE user_name = $1;