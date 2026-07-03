package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"remote-bot/internal/notifier"
	"remote-bot/internal/storage"
)

type Scheduler struct {
	cron     *cron.Cron
	requests *storage.RequestRepo
	dailies  *storage.DailyRepo
	notifier *notifier.Notifier
}

func New(requests *storage.RequestRepo, dailies *storage.DailyRepo, n *notifier.Notifier) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		requests: requests,
		dailies:  dailies,
		notifier: n,
	}
}

func (s *Scheduler) Start() error {
	_, err := s.cron.AddFunc("55 8 * * *", s.notifyRequests)
	if err != nil {
		return fmt.Errorf("add requests job: %w", err)
	}
	_, err = s.cron.AddFunc("* * * * *", s.notifyDailies)
	if err != nil {
		return fmt.Errorf("add dailies job: %w", err)
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) notifyRequests() {
	today := time.Now().Truncate(24 * time.Hour)
	requests, err := s.requests.PendingForDate(context.Background(), today)
	if err != nil {
		log.Printf("[scheduler] ошибка получения заявок: %v", err)
		return
	}
	if len(requests) == 0 {
		return
	}

	if err := s.notifier.NotifyGroupRequests(requests); err != nil {
		log.Printf("[scheduler] ошибка отправки уведомлений: %v", err)
		return
	}

	for _, req := range requests {
		if err := s.requests.MarkNotified(context.Background(), req.ID, req.Type); err != nil {
			log.Printf("[scheduler] ошибка пометки заявки id=%d: %v", req.ID, err)
		}
	}
}

func (s *Scheduler) notifyDailies() {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	timeStr := now.Format("15:04")
	dailies, err := s.dailies.PendingForDateTime(context.Background(), today, timeStr)
	if err != nil {
		log.Printf("[scheduler] ошибка получения дэйликов: %v", err)
		return
	}
	for _, d := range dailies {
		if err := s.notifier.NotifyGroupDaily(d); err != nil {
			log.Printf("[scheduler] ошибка отправки дэйлика id=%d: %v", d.ID, err)
			continue
		}
		if err := s.dailies.MarkNotified(context.Background(), d.ID); err != nil {
			log.Printf("[scheduler] ошибка пометки дэйлика id=%d: %v", d.ID, err)
		}
	}
}
