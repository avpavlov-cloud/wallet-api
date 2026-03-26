CREATE MATERIALIZED VIEW daily_volume_report AS
SELECT 
    a.currency,
    SUM(t.amount) as total_volume,
    COUNT(t.id) as transaction_count,
    NOW() as last_updated
FROM transactions t
JOIN accounts a ON t.from_account_id = a.id
WHERE t.created_at > NOW() - INTERVAL '24 hours'
GROUP BY a.currency;

-- Добавляем индекс для мгновенного чтения отчета
CREATE UNIQUE INDEX idx_daily_report_currency ON daily_volume_report (currency);
