CREATE TABLE IF NOT EXISTS orders (
    number       BIGINT PRIMARY KEY,
    user_id      BIGINT NOT NULL,
    status       VARCHAR(100) NOT NULL,
    accural      BIGINT,
    uploaded_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_orders_user
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
    );

CREATE INDEX IF NOT EXISTS orders_uploaded_at_idx
    ON orders (uploaded_at);
