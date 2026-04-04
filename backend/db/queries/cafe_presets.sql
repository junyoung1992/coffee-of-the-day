-- name: InsertCafePreset :exec
INSERT INTO cafe_presets (preset_id, cafe_name, coffee_name, tasting_tags)
VALUES (?, ?, ?, ?);

-- name: GetCafePresetByPresetID :one
SELECT preset_id, cafe_name, coffee_name, tasting_tags
FROM cafe_presets
WHERE preset_id = ?;

-- name: UpdateCafePreset :exec
UPDATE cafe_presets
SET cafe_name = ?, coffee_name = ?, tasting_tags = ?
WHERE preset_id = ?;
