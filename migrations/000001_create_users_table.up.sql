CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    login       VARCHAR(254) NOT NULL,
    password    VARCHAR(200) NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    current     BIGINT NOT NULL DEFAULT 0,
    withdrawn   BIGINT NOT NULL DEFAULT 0
    );

CREATE UNIQUE INDEX IF NOT EXISTS users_login_ui ON users(login);
CREATE INDEX IF NOT EXISTS users_created_at_idx ON users(created_at);