package domain

// RegisterRequest는 회원가입 요청 데이터이다.
type RegisterRequest struct {
	Email       string
	Password    string
	Username    string
	DisplayName string // 생략 시 Username으로 대체된다
}

// LoginRequest는 로그인 요청 데이터이다.
type LoginRequest struct {
	Email    string
	Password string
}

// AuthUser는 인증 후 클라이언트에 반환되는 사용자 정보이다.
// 비밀번호 해시 등 민감 정보는 포함하지 않는다.
type AuthUser struct {
	ID          string
	Email       string
	Username    string
	DisplayName string
}
