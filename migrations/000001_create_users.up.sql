BEGIN;

CREATE TABLE users
(
    id BIGSERIAL PRIMARY KEY,
    login varchar(50) NOT NULL UNIQUE,
	password varchar(255) NOT NULL
);

COMMIT;