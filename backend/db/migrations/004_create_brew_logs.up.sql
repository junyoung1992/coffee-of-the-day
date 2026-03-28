CREATE TABLE brew_logs (
    log_id          TEXT PRIMARY KEY REFERENCES coffee_logs(id) ON DELETE CASCADE,
    bean_name       TEXT NOT NULL,
    bean_origin     TEXT,
    bean_process    TEXT,
    roast_level     TEXT CHECK(roast_level IN ('light', 'medium', 'dark')),
    roast_date      TEXT,
    tasting_tags    TEXT NOT NULL DEFAULT '[]',
    tasting_note    TEXT,
    brew_method     TEXT NOT NULL CHECK(brew_method IN (
                        'pour_over', 'immersion', 'aeropress',
                        'espresso', 'moka_pot', 'siphon', 'cold_brew', 'other'
                    )),
    brew_device     TEXT,
    coffee_amount_g REAL,
    water_amount_ml REAL,
    water_temp_c    REAL,
    brew_time_sec   INTEGER,
    grind_size      TEXT,
    brew_steps      TEXT NOT NULL DEFAULT '[]',
    impressions     TEXT,
    rating          REAL CHECK(rating >= 0.5 AND rating <= 5.0)
);
