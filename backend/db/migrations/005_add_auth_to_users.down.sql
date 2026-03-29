DROP INDEX IF EXISTS idx_users_email;
ALTER TABLE users DROP COLUMN email;
ALTER TABLE users DROP COLUMN password_hash;
