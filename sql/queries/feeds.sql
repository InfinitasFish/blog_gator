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

-- name: GetFeedByUrl :one
SELECT * from feeds
WHERE feeds.feed_url = $1
LIMIT 1;

-- name: ListFeeds :many
SELECT users.user_name, feeds.feed_name, feeds.feed_url, feeds.user_id FROM feeds
INNER JOIN users ON feeds.user_id = users.id
ORDER BY feeds.feed_name;

-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = $2, updated_at = $2
WHERE feeds.id = $1;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;