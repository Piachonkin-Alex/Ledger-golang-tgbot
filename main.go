package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var host = os.Getenv("HOST")
var port = os.Getenv("PORT")
var user = os.Getenv("USER")
var password = os.Getenv("PASSWORD")
var dbname = os.Getenv("DBNAME")
var sslmode = os.Getenv("SSLMODE")
var token = os.Getenv("TOKEN")

var dbInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslmode)

func creator(l *Ledger, id ID, val Money) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := (*l).CreateAccount(ctx, id)
	if err != nil {
		return err
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = (*l).Deposit(ctx, id, val)
	return err
}

func messageHandler(update tgbotapi.Update, ledger Ledger, bot *tgbotapi.BotAPI) {
	if update.Message != nil { // If we got a message
		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi, "+update.Message.From.UserName+"! This is ledger tg-bot.")
			_, _ = bot.Send(msg)
		}
		args := strings.Split(update.Message.Text, " ")
		switch strings.ToLower(args[0]) {
		case "create_user":
			if len(args) != 3 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Write the id of new user")
				_, _ = bot.Send(msg)
			}
			val, e2 := strconv.Atoi(args[2])
			if e2 != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Id and deposit should be integer!")
				_, _ = bot.Send(msg)
			}
			err := creator(&ledger, ID(args[1]), Money(val))
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong happend!")
				_, _ = bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "User "+args[1]+" with balance "+args[2]+" added.")
				_, _ = bot.Send(msg)
			}
		case "get_balance":
			if len(args) != 2 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Write the id of user!")
				_, _ = bot.Send(msg)
			}
			mon, err := ledger.GetBalance(context.Background(), ID(args[1]))
			if err != nil {
				fmt.Println(err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong happend!")
				_, _ = bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The balance of user "+args[1]+" is "+strconv.Itoa(int(mon)))
				_, _ = bot.Send(msg)
			}
		case "table":
			users, err := ledger.GetTable(context.Background())
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong happend!")
				_, _ = bot.Send(msg)
			} else {
				data, _ := json.Marshal(users)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(data))
				_, _ = bot.Send(msg)
			}
		case "deposit":
			if len(args) != 3 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Write the id of new user and money")
				_, _ = bot.Send(msg)
			}
			amount, e2 := strconv.Atoi(args[2])
			if e2 != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Id and deposit should be integer!")
				_, _ = bot.Send(msg)
			}
			mon, err := ledger.Deposit(context.Background(), ID(args[1]), Money(amount))
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong happend!")
				_, _ = bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "User "+args[1]+" now with balance "+" "+strconv.Itoa(int(mon)))
				_, _ = bot.Send(msg)
			}

		case "withdraw":
			if len(args) != 3 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Write the id of new user and money")
				_, _ = bot.Send(msg)
			}
			amount, e2 := strconv.Atoi(args[2])
			if e2 != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Id and deposit should be integer!")
				_, _ = bot.Send(msg)
			}
			mon, err := ledger.Withdraw(context.Background(), ID(args[1]), Money(amount))
			if err != nil && errors.Is(ErrNoMoney, err) {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "User "+args[1]+" has not enough money.")
				_, _ = bot.Send(msg)
			} else if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong happend!")
				_, _ = bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "User "+args[1]+" now with balance "+" "+strconv.Itoa(int(mon)))
				_, _ = bot.Send(msg)
			}
		case "transfer":
			if len(args) != 4 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Write the id of new user and money")
				_, _ = bot.Send(msg)
			}
			amount, e2 := strconv.Atoi(args[3])
			if e2 != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Id and deposit should be integer!")
				_, _ = bot.Send(msg)
			}
			monFrom, monTo, err := ledger.Transfer(context.Background(), ID(args[1]), ID(args[2]), Money(amount))
			if err != nil && errors.Is(ErrNoMoney, err) {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "User "+args[1]+" has not enough money.")
				_, _ = bot.Send(msg)
			} else if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong happend!")
				_, _ = bot.Send(msg)
			} else {
				buffer := new(bytes.Buffer)
				buffer.Write([]byte("User "))
				buffer.Write([]byte(args[1]))
				buffer.Write([]byte(" now with balance " + strconv.Itoa(int(monFrom))))
				buffer.Write([]byte("\nUser "))
				buffer.Write([]byte(args[2]))
				buffer.Write([]byte(" now with balance " + strconv.Itoa(int(monTo))))
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, buffer.String())
				_, _ = bot.Send(msg)
			}

		}
	}
}

func runBot() {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	ctx := context.Background()
	ledger, err := New(ctx, dbInfo)

	if err != nil {
		fmt.Println(err)
		panic("gg")
	}

	wg := new(sync.WaitGroup)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
loop:
	for {
		select {
		case update := <-updates:
			go func() {
				wg.Add(1)
				messageHandler(update, ledger, bot)
				wg.Done()
			}()
		case <-stop:
			break loop
		}
	}
	wg.Wait()
}

func main() {
	time.Sleep(1 * time.Second)
	runBot()
}
