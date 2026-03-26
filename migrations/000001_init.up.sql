CREATE TABLE "accounts" (
  "id" BIGSERIAL PRIMARY KEY,
  "owner_name" varchar NOT NULL,
  "balance" bigint NOT NULL DEFAULT 0,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);
