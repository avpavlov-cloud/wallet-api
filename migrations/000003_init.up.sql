-- ИНДЕКСЫ ДЛЯ ПРОИЗВОДИТЕЛЬНОСТИ 
-- Ускоряет поиск всех транзакций конкретного пользователя
CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions(to_account_id);

-- Составной индекс (если часто ищем переводы между двумя конкретными людьми)
CREATE INDEX IF NOT EXISTS idx_transactions_pair ON transactions(from_account_id, to_account_id);