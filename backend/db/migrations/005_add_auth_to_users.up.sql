-- SQLite는 ALTER TABLE ADD COLUMN에서 UNIQUE 제약을 직접 지정할 수 없다.
-- UNIQUE 강제는 별도 인덱스로 처리한다.
ALTER TABLE users ADD COLUMN email TEXT;
ALTER TABLE users ADD COLUMN password_hash TEXT;

-- NULL 값은 UNIQUE 인덱스에서 중복으로 간주되지 않으므로 WHERE email IS NOT NULL 조건을 추가한다.
-- 기존 POC 시드 사용자(email=NULL)가 여러 명 있어도 충돌하지 않는다.
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
