package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/repository"
)

var (
	ErrEmailTaken         = errors.New("이미 사용 중인 이메일입니다")
	ErrInvalidCredentials = errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
	ErrInvalidToken       = errors.New("유효하지 않은 토큰입니다")
)

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 7 * 24 * time.Hour
)

// AuthTokens는 발급된 액세스/리프레시 토큰 쌍이다.
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
}

// AuthService는 회원가입·로그인·토큰 갱신 인터페이스를 정의한다.
type AuthService interface {
	Register(ctx context.Context, req domain.RegisterRequest) (domain.AuthUser, AuthTokens, error)
	Login(ctx context.Context, req domain.LoginRequest) (domain.AuthUser, AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (AuthTokens, error)
	GetUser(ctx context.Context, userID string) (domain.AuthUser, error)
}

// tokenClaims는 JWT 페이로드 구조다.
// token_type으로 액세스/리프레시 토큰을 구분하여 토큰 재사용 공격을 차단한다.
type tokenClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// DefaultAuthService는 AuthService의 기본 구현체다.
type DefaultAuthService struct {
	repo      repository.UserRepository
	jwtSecret []byte
	now       func() time.Time
}

// NewAuthService는 DefaultAuthService를 생성한다.
func NewAuthService(repo repository.UserRepository, jwtSecret string) *DefaultAuthService {
	return &DefaultAuthService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		now:       time.Now,
	}
}

// Register는 신규 사용자를 등록하고 즉시 로그인 토큰을 발급한다.
func (s *DefaultAuthService) Register(ctx context.Context, req domain.RegisterRequest) (domain.AuthUser, AuthTokens, error) {
	if err := validateRegisterRequest(req); err != nil {
		return domain.AuthUser{}, AuthTokens{}, err
	}

	// bcrypt는 비밀번호를 단방향 해싱하며, salt를 자동으로 포함한다.
	// DefaultCost(10)는 보안과 성능의 균형점이다.
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AuthUser{}, AuthTokens{}, fmt.Errorf("register: hash password: %w", err)
	}

	id, err := newUUID()
	if err != nil {
		return domain.AuthUser{}, AuthTokens{}, fmt.Errorf("register: generate id: %w", err)
	}

	displayName := req.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = req.Username
	}

	rec, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		ID:           id,
		Username:     req.Username,
		DisplayName:  displayName,
		Email:        req.Email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			return domain.AuthUser{}, AuthTokens{}, ErrEmailTaken
		}
		return domain.AuthUser{}, AuthTokens{}, fmt.Errorf("register: create user: %w", err)
	}

	tokens, err := s.generateTokens(rec.ID)
	if err != nil {
		return domain.AuthUser{}, AuthTokens{}, err
	}

	return recordToAuthUser(rec), tokens, nil
}

// Login은 이메일과 비밀번호를 검증하고 토큰을 발급한다.
func (s *DefaultAuthService) Login(ctx context.Context, req domain.LoginRequest) (domain.AuthUser, AuthTokens, error) {
	rec, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// 사용자가 없어도 잘못된 자격증명 오류를 반환해 이메일 존재 여부를 숨긴다
			return domain.AuthUser{}, AuthTokens{}, ErrInvalidCredentials
		}
		return domain.AuthUser{}, AuthTokens{}, fmt.Errorf("login: get user: %w", err)
	}

	// email/password가 없는 POC 시드 사용자는 새 인증 방식으로 로그인 불가
	if rec.PasswordHash == nil {
		return domain.AuthUser{}, AuthTokens{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*rec.PasswordHash), []byte(req.Password)); err != nil {
		return domain.AuthUser{}, AuthTokens{}, ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(rec.ID)
	if err != nil {
		return domain.AuthUser{}, AuthTokens{}, err
	}

	return recordToAuthUser(rec), tokens, nil
}

// Refresh는 유효한 리프레시 토큰을 검증하고 새 토큰 쌍을 발급한다.
func (s *DefaultAuthService) Refresh(ctx context.Context, refreshToken string) (AuthTokens, error) {
	userID, err := s.parseToken(refreshToken, "refresh")
	if err != nil {
		return AuthTokens{}, ErrInvalidToken
	}

	// 사용자가 여전히 존재하는지 확인 (계정 삭제 시 리프레시 토큰 무효화)
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return AuthTokens{}, ErrInvalidToken
	}

	return s.generateTokens(userID)
}

func (s *DefaultAuthService) generateTokens(userID string) (AuthTokens, error) {
	accessToken, err := s.signToken(userID, "access", s.now().Add(accessTokenDuration))
	if err != nil {
		return AuthTokens{}, fmt.Errorf("generate tokens: sign access token: %w", err)
	}

	refreshToken, err := s.signToken(userID, "refresh", s.now().Add(refreshTokenDuration))
	if err != nil {
		return AuthTokens{}, fmt.Errorf("generate tokens: sign refresh token: %w", err)
	}

	return AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *DefaultAuthService) signToken(userID, tokenType string, expiresAt time.Time) (string, error) {
	claims := tokenClaims{
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(s.now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *DefaultAuthService) parseToken(tokenStr, expectedType string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid claims")
	}

	if claims.TokenType != expectedType {
		return "", fmt.Errorf("expected token type %q, got %q", expectedType, claims.TokenType)
	}

	return claims.Subject, nil
}

func validateRegisterRequest(req domain.RegisterRequest) error {
	email := strings.TrimSpace(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		return &ValidationError{Field: "email", Message: "올바른 이메일 형식이 아닙니다"}
	}
	if len(req.Password) < 8 {
		return &ValidationError{Field: "password", Message: "비밀번호는 8자 이상이어야 합니다"}
	}
	if strings.TrimSpace(req.Username) == "" {
		return &ValidationError{Field: "username", Message: "사용자명이 필요합니다"}
	}
	return nil
}

// GetUser는 userID로 사용자 정보를 조회한다. /auth/me 엔드포인트에서 사용한다.
func (s *DefaultAuthService) GetUser(ctx context.Context, userID string) (domain.AuthUser, error) {
	rec, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return domain.AuthUser{}, err
	}
	return recordToAuthUser(rec), nil
}

func recordToAuthUser(rec repository.UserRecord) domain.AuthUser {
	email := ""
	if rec.Email != nil {
		email = *rec.Email
	}
	return domain.AuthUser{
		ID:          rec.ID,
		Email:       email,
		Username:    rec.Username,
		DisplayName: rec.DisplayName,
	}
}
