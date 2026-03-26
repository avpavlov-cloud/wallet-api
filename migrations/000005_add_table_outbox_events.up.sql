CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL, -- например, 'transfer_completed'
    payload JSONB NOT NULL,          -- данные для уведомления
    status VARCHAR(20) DEFAULT 'pending', -- pending, processed, failed
    created_at TIMESTAMPTZ DEFAULT NOW()
);
