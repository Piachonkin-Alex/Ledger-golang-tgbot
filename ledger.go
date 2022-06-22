package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"strings"
)

type MyLedger struct {
	db *sql.DB
}

func (l *MyLedger) CreateAccount(ctx context.Context, id ID) error {
	_, err := l.db.ExecContext(ctx, "INSERT INTO balances(id, money) VALUES($1, 0)", id)
	return err
}

func (l *MyLedger) GetBalance(ctx context.Context, id ID) (Money, error) {
	var balance int64
	err := l.db.QueryRowContext(ctx, "select money from balances where id = $1", id).Scan(&balance)
	return Money(balance), err
}
func (l *MyLedger) Deposit(ctx context.Context, id ID, amount Money) (Money, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	var curBalance int64
	err = tx.QueryRowContext(ctx, "select money from balances where id = $1 for update", id).Scan(&curBalance)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "update balances set money = $2 where id = $1", id, curBalance+int64(amount))
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return Money(curBalance + int64(amount)), nil
}
func (l *MyLedger) Withdraw(ctx context.Context, id ID, amount Money) (Money, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	var curBalance int64
	err = tx.QueryRowContext(ctx, "select money from balances where id = $1 for update", id).Scan(&curBalance)
	if err != nil {
		return 0, err
	}
	if int64(amount) > curBalance {
		return 0, ErrNoMoney
	}

	_, err = tx.ExecContext(ctx, "update balances set money = $2 where id = $1", id, curBalance-int64(amount))
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return Money(curBalance - int64(amount)), nil
}

func (l *MyLedger) Transfer(ctx context.Context, from, to ID, amount Money) (Money, Money, error) {
	if strings.Compare(string(from), string(to)) == 0 {
		return 0, 0, nil
	}

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	var curBalanceFrom int64
	var curBalanceTo int64

	if strings.Compare(string(from), string(to)) == -1 {
		err = tx.QueryRowContext(ctx, "select money from balances where id = $1 for update", from).Scan(&curBalanceFrom)
		if err != nil {
			return 0, 0, err
		}

		err = tx.QueryRowContext(ctx, "select money from balances where id = $1 for update", to).Scan(&curBalanceTo)
		if err != nil {
			return 0, 0, err
		}
	} else {
		err = tx.QueryRowContext(ctx, "select money from balances where id = $1 for update", to).Scan(&curBalanceTo)
		if err != nil {
			return 0, 0, err
		}

		err = tx.QueryRowContext(ctx, "select money from balances where id = $1 for update", from).Scan(&curBalanceFrom)
		if err != nil {
			return 0, 0, err
		}
	}

	if int64(amount) > curBalanceFrom {
		return 0, 0, ErrNoMoney
	}

	_, err = tx.ExecContext(ctx, "update balances set money = $2 where id = $1", from, curBalanceFrom-int64(amount))
	if err != nil {
		return 0, 0, err
	}

	_, err = tx.ExecContext(ctx, "update balances set money = $2 where id = $1", to, curBalanceTo+int64(amount))
	if err != nil {
		return 0, 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return Money(curBalanceFrom - int64(amount)), Money(curBalanceTo + int64(amount)), nil
}
func (l *MyLedger) Close() error {
	return l.db.Close()
}

func (l *MyLedger) GetTable(ctx context.Context) ([]User, error) {
	rows, err := l.db.QueryContext(ctx, "select id, money from balances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []User
	for rows.Next() {
		var id string
		var mon int
		if err := rows.Scan(&id, &mon); err != nil {
			return nil, err
		}
		res = append(res, User{ID(id), Money(mon)})
	}
	return res, nil
}

func New(ctx context.Context, dsn string) (Ledger, error) {
	fmt.Println("here")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, "create table if not exists balances (id text  PRIMARY KEY,\n money  integer  default null)")
	if err != nil {
		return nil, err
	}
	return &MyLedger{db: db}, nil
}
