package config

import "os"

type Config struct {
	Port          string
	DBPath        string
	POCSeedUserID string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "coffee.db"
	}

	pocSeedUserID := os.Getenv("POC_SEED_USER_ID")

	return Config{
		Port:          port,
		DBPath:        dbPath,
		POCSeedUserID: pocSeedUserID,
	}
}
