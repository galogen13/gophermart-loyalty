BEGIN;

DROP INDEX IF EXISTS users_login_password_idx;
DROP TABLE IF EXISTS users; 

COMMIT;