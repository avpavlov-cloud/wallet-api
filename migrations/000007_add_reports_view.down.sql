-- При удалении самого представления, связанные с ним индексы 
-- (например, idx_daily_report_currency) удаляются автоматически.
DROP MATERIALIZED VIEW IF EXISTS daily_volume_report;