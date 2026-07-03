package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tele "gopkg.in/telebot.v3"

	"remote-bot/internal/bot"
	"remote-bot/internal/config"
	"remote-bot/internal/notifier"
	"remote-bot/internal/scheduler"
	"remote-bot/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфиг: %v", err)
	}

	pool, err := storage.NewPostgres(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("БД: %v", err)
	}
	defer pool.Close()

	teleBot, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10},
	})
	if err != nil {
		log.Fatalf("telebot: %v", err)
	}

	repos := bot.Repos{
		Employees:    storage.NewEmployeeRepo(pool),
		Requests:     storage.NewRequestRepo(pool),
		Dailies:      storage.NewDailyRepo(pool),
		DialogStates: storage.NewDialogStateRepo(pool),
	}

	n := notifier.New(teleBot, cfg)

	b, err := bot.New(cfg, repos, n, teleBot)
	if err != nil {
		log.Fatalf("бот: %v", err)
	}

	sched := scheduler.New(repos.Requests, repos.Dailies, n, cfg.AdminID)
	if err := sched.Start(); err != nil {
		log.Fatalf("планировщик: %v", err)
	}
	defer sched.Stop()

	go b.Start()
	log.Println("бот запущен")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("завершение работы...")
	b.Stop()
}
