-- token_version은 로그아웃 시 증가시켜 이전에 발급된 리프레시 토큰을 일괄 무효화한다.
-- JWT 클레임의 token_version이 DB 값과 다르면 서버가 토큰을 거부한다.
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
