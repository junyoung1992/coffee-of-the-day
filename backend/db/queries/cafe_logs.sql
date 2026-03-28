-- name: InsertCafeLog :exec
INSERT INTO cafe_logs (log_id, cafe_name, location, coffee_name, bean_origin, bean_process, roast_level, tasting_tags, tasting_note, impressions, rating)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCafeLogByLogID :one
SELECT log_id, cafe_name, location, coffee_name, bean_origin, bean_process, roast_level, tasting_tags, tasting_note, impressions, rating
FROM cafe_logs
WHERE log_id = ?;

-- name: UpdateCafeLog :exec
UPDATE cafe_logs
SET cafe_name = ?, location = ?, coffee_name = ?, bean_origin = ?, bean_process = ?, roast_level = ?, tasting_tags = ?, tasting_note = ?, impressions = ?, rating = ?
WHERE log_id = ?;
