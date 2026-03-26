ALTER TABLE transactions ADD COLUMN idempotency_key UUID UNIQUE;
