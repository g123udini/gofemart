CREATE TABLE IF NOT EXISTS withdrawals (
    number       BIGINT PRIMARY KEY,
    user_id      BIGINT NOT NULL,
    sum          BIGINT NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_orders_user
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
    );

CREATE INDEX IF NOT EXISTS withdrawals_processed_at_at_idx
    ON withdrawals (processed_at);