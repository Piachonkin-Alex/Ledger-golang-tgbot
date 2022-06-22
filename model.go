package main

import (
	"context"
	"errors"
)

type (
	ID    string
	Money int64
)

type User struct {
	Id  ID    `json:"id"`
	Mon Money `json:"money"`
}

var ErrNoMoney = errors.New("no money")

type Ledger interface {
	CreateAccount(ctx context.Context, id ID) error
	GetBalance(ctx context.Context, id ID) (Money, error)
	Deposit(ctx context.Context, id ID, amount Money) (Money, error)
	Withdraw(ctx context.Context, id ID, amount Money) (Money, error)
	Transfer(ctx context.Context, from, to ID, amount Money) (Money, Money, error)
	GetTable(ctx context.Context) ([]User, error)
	Close() error
}
