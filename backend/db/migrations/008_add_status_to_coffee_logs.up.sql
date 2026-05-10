-- status는 로그의 완성 단계를 나타낸다. 'draft'는 임시 저장 상태(필수 필드 미충족 허용),
-- 'published'는 정식 발행 상태. 기존 row는 모두 완성된 로그이므로 DEFAULT로 backfill한다.
ALTER TABLE coffee_logs
  ADD COLUMN status TEXT NOT NULL DEFAULT 'published'
  CHECK(status IN ('draft', 'published'));
