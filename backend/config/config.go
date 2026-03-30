package config

import (
	"errors"
	"os"
)

// minJWTSecretLen은 JWT 시크릿의 최소 길이다.
// 256비트(32바이트) 이상이어야 충분한 엔트로피를 보장한다.
const minJWTSecretLen = 32

type Config struct {
	Port          string
	DBPath        string
	POCSeedUserID string
	JWTSecret     string
	IsProduction  bool
}

// Load는 환경변수에서 설정을 읽어 Config를 반환한다.
// 운영 환경에서 JWT_SECRET이 없거나 너무 짧으면 에러를 반환해 서버 시작을 막는다.
func Load() (Config, error) {
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

	isProduction := os.Getenv("GO_ENV") == "production"

	jwtSecret, err := loadJWTSecret(isProduction)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:          port,
		DBPath:        dbPath,
		POCSeedUserID: pocSeedUserID,
		JWTSecret:     jwtSecret,
		IsProduction:  isProduction,
	}, nil
}

// loadJWTSecret은 JWT_SECRET 환경변수를 읽는다.
// 운영 환경에서는 미설정 또는 너무 짧은 시크릿을 허용하지 않는다.
// 개발 환경에서는 fallback 값을 사용한다.
func loadJWTSecret(isProduction bool) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	if isProduction {
		if secret == "" {
			return "", errors.New("JWT_SECRET이 설정되지 않았습니다: 운영 환경에서는 반드시 설정해야 합니다")
		}
		if len(secret) < minJWTSecretLen {
			return "", errors.New("JWT_SECRET이 너무 짧습니다: 최소 32바이트 이상이어야 합니다")
		}
		return secret, nil
	}

	// 개발 환경: 명시적으로 설정된 경우 그 값을 사용하고, 없으면 dev fallback을 사용한다.
	if secret != "" {
		return secret, nil
	}
	return "dev-secret-change-in-production-must-be-32b", nil
}
