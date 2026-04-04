CREATE TABLE presets (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    name         TEXT NOT NULL,
    log_type     TEXT NOT NULL CHECK(log_type IN ('cafe', 'brew')),
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE cafe_presets (
    preset_id    TEXT PRIMARY KEY REFERENCES presets(id) ON DELETE CASCADE,
    cafe_name    TEXT NOT NULL,
    coffee_name  TEXT NOT NULL,
    tasting_tags TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE brew_presets (
    preset_id     TEXT PRIMARY KEY REFERENCES presets(id) ON DELETE CASCADE,
    bean_name     TEXT NOT NULL,
    brew_method   TEXT NOT NULL CHECK(brew_method IN (
                      'pour_over', 'immersion', 'aeropress',
                      'espresso', 'moka_pot', 'siphon', 'cold_brew', 'other'
                  )),
    recipe_detail TEXT,
    brew_steps    TEXT NOT NULL DEFAULT '[]'
);
