package config

import "os"

type Config struct {
	Port          string
	DBPath        string
	POCSeedUserID string
	JWTSecret     string
	IsProduction  bool
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
	if pocSeedUserID == "" {
		pocSeedUserID = "dev-user"
	}

	// JWT_SECRET은 프로덕션에서 반드시 강력한 랜덤 값으로 교체해야 한다.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	return Config{
		Port:          port,
		DBPath:        dbPath,
		POCSeedUserID: pocSeedUserID,
		JWTSecret:     jwtSecret,
		IsProduction:  os.Getenv("GO_ENV") == "production",
	}
}
