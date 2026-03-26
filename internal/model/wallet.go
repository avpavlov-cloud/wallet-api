package model

import "time"

// Account представляет запись в таблице accounts
type Account struct {
	ID        int64     `json:"id"`
	OwnerName string    `json:"owner_name"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAccountRequest для валидации входящего JSON при создании
type CreateAccountRequest struct {
	OwnerName string  `json:"owner_name" binding:"required"`
	Currency  string  `json:"currency" binding:"required,oneof=USD EUR RUB"`
	Balance   float64 `json:"balance" binding:"required,gte=0"`
}

// TransferRequest для валидации перевода
type TransferRequest struct {
	FromAccountID int64   `json:"from_account_id" binding:"required"`
	ToAccountID   int64   `json:"to_account_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}
