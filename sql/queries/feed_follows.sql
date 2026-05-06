-- name: CreateFeedFollow :one
WITH insert_feed_follow AS (
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)
SELECT insert_feed_follow.*, users.user_name, feeds.feed_name FROM insert_feed_follow
INNER JOIN users ON users.id = insert_feed_follow.user_id
INNER JOIN feeds ON feeds.id = insert_feed_follow.feed_id;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE $1 = feed_follows.user_id AND $2 = feed_follows.feed_id;