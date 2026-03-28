CREATE TABLE cafe_logs (
    log_id       TEXT PRIMARY KEY REFERENCES coffee_logs(id) ON DELETE CASCADE,
    cafe_name    TEXT NOT NULL,
    location     TEXT,
    coffee_name  TEXT NOT NULL,
    bean_origin  TEXT,
    bean_process TEXT,
    roast_level  TEXT CHECK(roast_level IN ('light', 'medium', 'dark')),
    tasting_tags TEXT NOT NULL DEFAULT '[]',
    tasting_note TEXT,
    impressions  TEXT,
    rating       REAL CHECK(rating >= 0.5 AND rating <= 5.0)
);
