-- 1. Создаем таблицу транзакций
CREATE TABLE "transactions" (
  "id" BIGSERIAL PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_account_id" bigint NOT NULL,
  "amount" numeric(10, 2) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  
  -- Внешние ключи для связи с таблицей accounts
  FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id"),
  FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id")
);

-- 2. Добавляем проверку на неотрицательный баланс в таблицу accounts
-- Это заставит tx.Exec вернуть ошибку, если денег на счету не хватает
ALTER TABLE "accounts" ADD CONSTRAINT "positive_balance" CHECK (balance >= 0);
