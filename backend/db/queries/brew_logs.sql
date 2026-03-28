-- name: InsertBrewLog :exec
INSERT INTO brew_logs (log_id, bean_name, bean_origin, bean_process, roast_level, roast_date, tasting_tags, tasting_note, brew_method, brew_device, coffee_amount_g, water_amount_ml, water_temp_c, brew_time_sec, grind_size, brew_steps, impressions, rating)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetBrewLogByLogID :one
SELECT log_id, bean_name, bean_origin, bean_process, roast_level, roast_date, tasting_tags, tasting_note, brew_method, brew_device, coffee_amount_g, water_amount_ml, water_temp_c, brew_time_sec, grind_size, brew_steps, impressions, rating
FROM brew_logs
WHERE log_id = ?;

-- name: UpdateBrewLog :exec
UPDATE brew_logs
SET bean_name = ?, bean_origin = ?, bean_process = ?, roast_level = ?, roast_date = ?, tasting_tags = ?, tasting_note = ?, brew_method = ?, brew_device = ?, coffee_amount_g = ?, water_amount_ml = ?, water_temp_c = ?, brew_time_sec = ?, grind_size = ?, brew_steps = ?, impressions = ?, rating = ?
WHERE log_id = ?;
