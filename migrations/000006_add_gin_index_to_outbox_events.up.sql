-- migrations/000003_add_jsonb_index.up.sql

-- GIN индекс позволяет эффективно искать ключи и значения внутри JSONB
CREATE INDEX idx_outbox_payload_gin ON outbox_events USING GIN (payload);

-- Специальный индекс для поиска по конкретному полю внутри JSON (jsonb_path_ops)
-- Работает еще быстрее, если мы ищем точное совпадение ключа
CREATE INDEX idx_outbox_payload_path ON outbox_events USING GIN (payload jsonb_path_ops);
