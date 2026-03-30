package config

import (
	"strings"
	"testing"
)

func TestLoadJWTSecret_Production(t *testing.T) {
	t.Run("시크릿 미설정 시 에러", func(t *testing.T) {
		_, err := loadJWTSecret(true)
		if err == nil {
			t.Fatal("에러가 반환되어야 하지만 nil이 반환됐습니다")
		}
	})

	t.Run("시크릿이 너무 짧으면 에러", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "short")
		_, err := loadJWTSecret(true)
		if err == nil {
			t.Fatal("에러가 반환되어야 하지만 nil이 반환됐습니다")
		}
	})

	t.Run("32바이트 이상이면 성공", func(t *testing.T) {
		secret := strings.Repeat("a", minJWTSecretLen)
		t.Setenv("JWT_SECRET", secret)
		got, err := loadJWTSecret(true)
		if err != nil {
			t.Fatalf("예상치 않은 에러: %v", err)
		}
		if got != secret {
			t.Errorf("expected %q, got %q", secret, got)
		}
	})
}

func TestLoadJWTSecret_Development(t *testing.T) {
	t.Run("미설정 시 dev fallback 반환", func(t *testing.T) {
		got, err := loadJWTSecret(false)
		if err != nil {
			t.Fatalf("예상치 않은 에러: %v", err)
		}
		if got == "" {
			t.Error("fallback 값이 비어 있습니다")
		}
	})

	t.Run("명시적으로 설정된 경우 해당 값 반환", func(t *testing.T) {
		secret := "custom-dev-secret"
		t.Setenv("JWT_SECRET", secret)
		got, err := loadJWTSecret(false)
		if err != nil {
			t.Fatalf("예상치 않은 에러: %v", err)
		}
		if got != secret {
			t.Errorf("expected %q, got %q", secret, got)
		}
	})
}

func TestLoad_ProductionMissingSecret(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	// JWT_SECRET 미설정 상태에서 Load 호출
	_, err := Load()
	if err == nil {
		t.Fatal("운영 환경에서 JWT_SECRET 없이 Load가 성공하면 안 됩니다")
	}
}

func TestLoad_DevelopmentDefaults(t *testing.T) {
	_, err := Load()
	if err != nil {
		t.Fatalf("개발 환경 기본값으로 Load가 실패하면 안 됩니다: %v", err)
	}
}
