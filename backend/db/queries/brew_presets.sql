-- name: InsertBrewPreset :exec
INSERT INTO brew_presets (preset_id, bean_name, brew_method, recipe_detail, brew_steps)
VALUES (?, ?, ?, ?, ?);

-- name: GetBrewPresetByPresetID :one
SELECT preset_id, bean_name, brew_method, recipe_detail, brew_steps
FROM brew_presets
WHERE preset_id = ?;

-- name: UpdateBrewPreset :exec
UPDATE brew_presets
SET bean_name = ?, brew_method = ?, recipe_detail = ?, brew_steps = ?
WHERE preset_id = ?;
