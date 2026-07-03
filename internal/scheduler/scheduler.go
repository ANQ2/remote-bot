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
	adminID  int64
}

func New(requests *storage.RequestRepo, dailies *storage.DailyRepo, n *notifier.Notifier, adminID int64) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		requests: requests,
		dailies:  dailies,
		notifier: n,
		adminID:  adminID,
	}
}

func (s *Scheduler) heartbeat() {
	now := time.Now().Format("02.01.2006 15:04:05")
	err := s.notifier.SendToUser(s.adminID, fmt.Sprintf("✅ Бот работает нормально\n🕐 %s", now))
	if err != nil {
		log.Printf("[heartbeat] ошибка отправки: %v", err)
	}
}

func (s *Scheduler) Start() error {
	_, err := s.cron.AddFunc("0 8 * * *", s.notifyRequests)
	if err != nil {
		return fmt.Errorf("add requests job: %w", err)
	}
	_, err = s.cron.AddFunc("* * * * *", s.notifyDailies)
	if err != nil {
		return fmt.Errorf("add dailies job: %w", err)
	}
	_, err = s.cron.AddFunc(" 0 * * * *", s.heartbeat)
	if err != nil {
		return fmt.Errorf("add heartbeat job: %w", err)
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
		msg := fmt.Sprintf("❌ Ошибка получения заявок: %v", err)
		log.Printf("[scheduler] %s", msg)
		_ = s.notifier.SendToUser(s.adminID, msg)
		return
	}
	if len(requests) == 0 {
		return
	}
	if err := s.notifier.NotifyGroupRequests(requests); err != nil {
		msg := fmt.Sprintf("❌ Ошибка отправки уведомлений: %v", err)
		log.Printf("[scheduler] %s", msg)
		_ = s.notifier.SendToUser(s.adminID, msg)
		return
	}
	for _, req := range requests {
		if err := s.requests.MarkNotified(context.Background(), req.ID, req.Type); err != nil {
			msg := fmt.Sprintf("❌ Ошибка пометки заявки id=%d: %v", req.ID, err)
			log.Printf("[scheduler] %s", msg)
			_ = s.notifier.SendToUser(s.adminID, msg)
		}
	}
}

func (s *Scheduler) notifyDailies() {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	timeStr := now.Format("15:04")
	dailies, err := s.dailies.PendingForDateTime(context.Background(), today, timeStr)
	if err != nil {
		msg := fmt.Sprintf("❌ Ошибка получения дэйликов: %v", err)
		log.Printf("[scheduler] %s", msg)
		_ = s.notifier.SendToUser(s.adminID, msg)
		return
	}
	for _, d := range dailies {
		if err := s.notifier.NotifyGroupDaily(d); err != nil {
			msg := fmt.Sprintf("❌ Ошибка отправки дэйлика id=%d: %v", d.ID, err)
			log.Printf("[scheduler] %s", msg)
			_ = s.notifier.SendToUser(s.adminID, msg)
			continue
		}
		if err := s.dailies.MarkNotified(context.Background(), d.ID); err != nil {
			msg := fmt.Sprintf("❌ Ошибка пометки дэйлика id=%d: %v", d.ID, err)
			log.Printf("[scheduler] %s", msg)
			_ = s.notifier.SendToUser(s.adminID, msg)
		}
	}
}
