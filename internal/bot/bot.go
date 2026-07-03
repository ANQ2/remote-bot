package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"remote-bot/internal/config"
	"remote-bot/internal/domain"
	"remote-bot/internal/notifier"
	"remote-bot/internal/storage"
)

type Repos struct {
	Employees    *storage.EmployeeRepo
	Requests     *storage.RequestRepo
	Dailies      *storage.DailyRepo
	DialogStates *storage.DialogStateRepo
}

type Bot struct {
	tele      *tele.Bot
	cfg       *config.Config
	notifier  *notifier.Notifier
	employees *storage.EmployeeRepo
	requests  *storage.RequestRepo
	dailies   *storage.DailyRepo
	states    *storage.DialogStateRepo
}

func New(cfg *config.Config, repos Repos, n *notifier.Notifier, teleBot *tele.Bot) (*Bot, error) {
	b := &Bot{
		tele:      teleBot,
		cfg:       cfg,
		notifier:  n,
		employees: repos.Employees,
		requests:  repos.Requests,
		dailies:   repos.Dailies,
		states:    repos.DialogStates,
	}
	b.registerHandlers()
	return b, nil
}

func (b *Bot) Tele() *tele.Bot {
	return b.tele
}

func (b *Bot) Start() {
	b.tele.Start()
}

func (b *Bot) Stop() {
	b.tele.Stop()
}

func (b *Bot) registerHandlers() {
	b.tele.Handle("/start", b.handleStart)
	b.tele.Handle("/request", b.handleRequest)
	b.tele.Handle("/cancel", b.handleCancel)
	b.tele.Handle("/myid", b.handleMyID)
	b.tele.Handle("/daily", b.handleDaily, onlyPM(b.cfg))

	b.tele.Handle(&btnRemote, b.handleBtnRemote)
	b.tele.Handle(&btnSick, b.handleBtnSick)
	b.tele.Handle(&btnDailyShort, b.handleBtnDailyShort)
	b.tele.Handle(&btnOnline, b.handleBtnOnline)
	b.tele.Handle(&btnOffline, b.handleBtnOffline)
	b.tele.Handle(&btnConfirmSend, b.handleBtnConfirmSend)
	b.tele.Handle(&btnEditTime, b.handleBtnEditTime)
	b.tele.Handle(&btnEditLoc, b.handleBtnEditLocation)
	b.tele.Handle(&btnUseLastLoc, b.handleBtnUseLastLoc)

	// Регистрируем кнопки дат на 2 недели вперёд
	b.registerDateButtons()

	// Регистрируем кнопки времени с 09:00 до 18:00
	b.registerTimeButtons()

	b.tele.Handle(tele.OnAddedToGroup, b.handleAddedToGroup)
	b.tele.Handle(tele.OnText, b.handleText)
}

func (b *Bot) registerDateButtons() {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for week := 0; week < 2; week++ {
		for day := 0; day < 5; day++ {
			date := monday.AddDate(0, 0, week*7+day)
			if date.Before(today) {
				continue
			}
			dateStr := date.Format("2006-01-02")
			unique := fmt.Sprintf("date_%s", dateStr)
			btn := tele.Btn{Unique: unique}
			captured := dateStr
			b.tele.Handle(&btn, func(c tele.Context) error {
				return b.handleDateCallback(c, captured)
			})
		}
	}
}

func (b *Bot) registerTimeButtons() {
	for h := 9; h <= 18; h++ {
		for _, m := range []int{0, 30} {
			if h == 18 && m == 30 {
				break
			}
			timeStr := fmt.Sprintf("%02d:%02d", h, m)
			unique := fmt.Sprintf("time_%s", strings.ReplaceAll(timeStr, ":", "_"))
			btn := tele.Btn{Unique: unique}
			captured := timeStr
			b.tele.Handle(&btn, func(c tele.Context) error {
				return b.handleTimeCallback(c, captured)
			})
		}
	}
}

func (b *Bot) getState(c tele.Context) (*domain.DialogState, error) {
	return b.states.Get(context.Background(), c.Sender().ID)
}

func (b *Bot) setState(telegramID int64, step domain.DialogStep, payload map[string]string) error {
	return b.states.Set(context.Background(), telegramID, step, payload)
}

func (b *Bot) resetState(telegramID int64) error {
	return b.states.Reset(context.Background(), telegramID)
}
