package bot

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"

	"remote-bot/internal/domain"
)

func (b *Bot) handleDaily(c tele.Context) error {
	if err := b.setState(c.Sender().ID, domain.StepPMAwaitDate, nil); err != nil {
		return err
	}
	return c.Send("📅 Выбери дату дэйлика:", weekDatesKeyboard())
}

func (b *Bot) handleBtnDailyShort(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepAwaitType {
		return c.Respond()
	}
	if !b.cfg.IsPM(c.Sender().ID) {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)
	return b.handleDaily(c)
}

func (b *Bot) handleDailyDate(c tele.Context, state *domain.DialogState) error {
	return c.Send("📅 Выбери дату дэйлика:", weekDatesKeyboard())
}

func (b *Bot) handleDailyTime(c tele.Context, state *domain.DialogState) error {
	return c.Send("🕐 Выбери время дэйлика:", timeKeyboard())
}

func (b *Bot) handleBtnOnline(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMAwaitMode {
		return c.Respond()
	}
	return b.handleModeChosen(c, domain.DailyOnline, state)
}

func (b *Bot) handleBtnOffline(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMAwaitMode {
		return c.Respond()
	}
	return b.handleModeChosen(c, domain.DailyOffline, state)
}

func (b *Bot) handleModeChosen(c tele.Context, mode domain.DailyMode, state *domain.DialogState) error {
	_ = c.Edit(c.Message().Text)

	payload := map[string]string{
		"date": state.Payload["date"],
		"time": state.Payload["time"],
		"mode": string(mode),
	}
	if err := b.setState(c.Sender().ID, domain.StepPMAwaitLocation, payload); err != nil {
		return err
	}

	var prompt string
	if mode == domain.DailyOnline {
		prompt = "🔗 Введи ссылку на встречу (Zoom, Meet, Teams и т.д.)"
	} else {
		prompt = "📍 Введи адрес офиса для дэйлика"
	}

	last, _ := b.dailies.GetLastByMode(context.Background(), mode)
	if last != nil && last.Location != "" {
		var lastLabel string
		if mode == domain.DailyOnline {
			lastLabel = fmt.Sprintf("Прошлая ссылка: %s", last.Location)
		} else {
			lastLabel = fmt.Sprintf("Прошлый адрес: %s", last.Location)
		}
		return c.Send(fmt.Sprintf("%s\n\n%s", prompt, lastLabel), locationKeyboard(true))
	}
	return c.Send(prompt)
}

func (b *Bot) handleDailyLocation(c tele.Context, state *domain.DialogState) error {
	return b.saveDailyLocation(c, state, c.Text())
}

func (b *Bot) handleBtnUseLastLoc(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMAwaitLocation {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)

	mode := domain.DailyMode(state.Payload["mode"])
	last, err := b.dailies.GetLastByMode(context.Background(), mode)
	if err != nil || last == nil {
		return c.Send("⚠️ Не удалось найти прошлый адрес/ссылку. Введи вручную.")
	}
	return b.saveDailyLocation(c, state, last.Location)
}

func (b *Bot) saveDailyLocation(c tele.Context, state *domain.DialogState, location string) error {
	payload := map[string]string{
		"date":     state.Payload["date"],
		"time":     state.Payload["time"],
		"mode":     state.Payload["mode"],
		"location": location,
	}
	if err := b.setState(c.Sender().ID, domain.StepPMConfirm, payload); err != nil {
		return err
	}

	date, _ := time.Parse("2006-01-02", payload["date"])
	mode := domain.DailyMode(payload["mode"])

	var modeLabel, locLabel string
	if mode == domain.DailyOnline {
		modeLabel = "онлайн 💻"
		locLabel = "Ссылка"
	} else {
		modeLabel = "офлайн 🏢"
		locLabel = "Адрес"
	}

	today := time.Now().Truncate(24 * time.Hour)
	var dayLabel string
	if date.Equal(today) {
		dayLabel = "сегодня"
	} else if date.Equal(today.Add(24 * time.Hour)) {
		dayLabel = "завтра"
	}

	var dateStr string
	if dayLabel != "" {
		dateStr = fmt.Sprintf("%s (%s)", date.Format("02.01.2006"), dayLabel)
	} else {
		dateStr = date.Format("02.01.2006")
	}

	card := fmt.Sprintf(
		"📋 Дэйлик %s\n%s\n\n%s: %s\nВремя: %s\n\nОтправить уведомление команде?",
		modeLabel, dateStr, locLabel, location, payload["time"],
	)
	return c.Send(card, confirmKeyboard())
}

func (b *Bot) handleBtnConfirmSend(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMConfirm {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)

	date, err := time.Parse("2006-01-02", state.Payload["date"])
	if err != nil {
		_ = b.resetState(c.Sender().ID)
		return c.Send("⚠️ Ошибка даты, начни заново: /daily")
	}

	mode := domain.DailyMode(state.Payload["mode"])
	timeStr := state.Payload["time"]
	location := state.Payload["location"]

	daily, err := b.dailies.Create(context.Background(), date, timeStr, mode, location, c.Sender().ID)
	if err != nil {
		return fmt.Errorf("create daily: %w", err)
	}

	if err := b.resetState(c.Sender().ID); err != nil {
		return err
	}

	if err := b.notifier.NotifyGroupDaily(*daily); err != nil {
		return fmt.Errorf("notify group: %w", err)
	}

	return c.Send("✅ Уведомление отправлено команде!", cancelLastDailyKeyboard())
}

func (b *Bot) handleBtnCancelLastDaily(c tele.Context) error {
	_ = c.Edit(c.Message().Text)
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}
	daily, err := b.dailies.DeleteLastByCreator(context.Background(), sender.ID)
	if err != nil {
		return c.Send("❌ Не удалось отменить дэйлик.")
	}
	_ = b.notifier.NotifyGroupCancelDaily(fullName, sender.Username, *daily)
	return c.Send("✅ Дэйлик отменён. Команда получила уведомление.")
}

func (b *Bot) handleBtnEditTime(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMConfirm {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)

	payload := map[string]string{
		"date":     state.Payload["date"],
		"mode":     state.Payload["mode"],
		"location": state.Payload["location"],
	}
	if err := b.setState(c.Sender().ID, domain.StepPMAwaitTime, payload); err != nil {
		return err
	}
	return c.Send("🕐 Выбери новое время:", timeKeyboard())
}

func (b *Bot) handleBtnEditLocation(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMConfirm {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)

	payload := map[string]string{
		"date": state.Payload["date"],
		"time": state.Payload["time"],
		"mode": state.Payload["mode"],
	}
	if err := b.setState(c.Sender().ID, domain.StepPMAwaitLocation, payload); err != nil {
		return err
	}

	mode := domain.DailyMode(state.Payload["mode"])
	if mode == domain.DailyOnline {
		return c.Send("🔗 Введи новую ссылку на встречу")
	}
	return c.Send("📍 Введи новый адрес")
}
