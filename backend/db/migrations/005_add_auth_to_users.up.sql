-- email, password_hash 컬럼 추가
-- ALTER TABLE ADD COLUMN은 NOT NULL DEFAULT 없이 nullable로만 추가 가능 (SQLite 제약)
-- 기존 POC 사용자 행은 NULL 값을 가지며, 신규 회원가입 시에만 두 컬럼이 채워진다.
ALTER TABLE users ADD COLUMN email TEXT UNIQUE;
ALTER TABLE users ADD COLUMN password_hash TEXT;
