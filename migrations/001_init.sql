CREATE TABLE IF NOT EXISTS transfers (
    id SERIAL PRIMARY KEY,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,
    value NUMERIC NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transfers_block_number ON transfers(block_number);
CREATE INDEX idx_transfers_from_address ON transfers(from_address);
CREATE INDEX idx_transfers_to_address ON transfers(to_address); 