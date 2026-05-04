-- name: AddFeed :one
INSERT INTO feeds (id, created_at, updated_at, feed_name, feed_url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: ListFeeds :many
SELECT users.user_name, feeds.feed_name, feeds.feed_url, feeds.user_id FROM feeds
INNER JOIN users ON feeds.user_id = users.id
ORDER BY feeds.feed_name;