BEGIN;

CREATE TABLE withdrawals
(
    id BIGSERIAL PRIMARY KEY,
    order_number VARCHAR(50) NOT NULL UNIQUE,
    sum DECIMAL(12,2),
    processed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    user_id BIGINT NOT NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_withdrawals_user_id ON withdrawals(user_id);

COMMIT;