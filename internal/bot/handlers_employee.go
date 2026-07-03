package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	tele "gopkg.in/telebot.v3"

	"remote-bot/internal/domain"
)

func (b *Bot) handleStart(c tele.Context) error {
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}
	_, err := b.employees.GetOrCreate(context.Background(), sender.ID, sender.Username, fullName)
	if err != nil {
		return fmt.Errorf("getOrCreate employee: %w", err)
	}
	return c.Send(
		"👋 Привет! Я помогу оформить удалёнку или больничный.\n\n" +
			"Выбери команду:\n" +
			"/request — подать заявку\n" +
			"/cancel — отменить текущий диалог",
	)
}

func (b *Bot) handleRequest(c tele.Context) error {
	if err := b.setState(c.Sender().ID, domain.StepAwaitType, nil); err != nil {
		return err
	}
	isPM := b.cfg.IsPM(c.Sender().ID)
	return c.Send("Выбери тип заявки:", requestTypeKeyboard(isPM))
}

func (b *Bot) handleCancel(c tele.Context) error {
	if err := b.resetState(c.Sender().ID); err != nil {
		return err
	}
	return c.Send("❌ Диалог отменён.")
}

func (b *Bot) handleBtnRemote(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepAwaitType {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)
	payload := map[string]string{"type": string(domain.RequestRemote)}
	if err := b.setState(c.Sender().ID, domain.StepAwaitDate, payload); err != nil {
		return err
	}
	return c.Send("📅 Выбери дату удалёнки:", weekDatesKeyboard())
}

func (b *Bot) handleBtnSick(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepAwaitType {
		return c.Respond()
	}
	_ = c.Edit(c.Message().Text)
	if err := b.setState(c.Sender().ID, domain.StepAwaitSickFrom, nil); err != nil {
		return err
	}
	return c.Send("📅 Выбери дату начала больничного:", weekDatesKeyboard())
}

// handleDateCallback обрабатывает нажатие кнопки с датой.
func (b *Bot) handleDateCallback(c tele.Context, dateStr string) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка даты"})
	}

	_ = c.Edit(c.Message().Text)

	switch state.Step {
	case domain.StepAwaitDate:
		// Удалёнка — сохраняем и подтверждаем
		sender := c.Sender()
		fullName := sender.FirstName
		if sender.LastName != "" {
			fullName += " " + sender.LastName
		}
		emp, err := b.employees.GetOrCreate(context.Background(), sender.ID, sender.Username, fullName)
		if err != nil {
			return err
		}
		if _, err := b.requests.CreateRemote(context.Background(), emp.ID, date); err != nil {
			return err
		}
		if err := b.resetState(sender.ID); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"✅ Заявка принята!\nТип: удалёнка\nДата: %s\n\nВ 8:00 утра команда получит уведомление.",
			date.Format("02.01.2006"),
		))

	case domain.StepAwaitSickFrom:
		// Больничный — запоминаем дату начала, просим дату конца
		payload := map[string]string{"date_from": dateStr}
		if err := b.setState(c.Sender().ID, domain.StepAwaitSickTo, payload); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"📅 Дата начала: %s\nТеперь выбери дату окончания больничного:",
			date.Format("02.01.2006"),
		), weekDatesKeyboard())

	case domain.StepAwaitSickTo:
		// Больничный — сохраняем
		dateFrom, err := time.Parse("2006-01-02", state.Payload["date_from"])
		if err != nil {
			_ = b.resetState(c.Sender().ID)
			return c.Send("⚠️ Ошибка, начни заново: /request")
		}
		if date.Before(dateFrom) {
			return c.Send("⚠️ Дата окончания не может быть раньше даты начала. Выбери снова:", weekDatesKeyboard())
		}
		sender := c.Sender()
		fullName := sender.FirstName
		if sender.LastName != "" {
			fullName += " " + sender.LastName
		}
		emp, err := b.employees.GetOrCreate(context.Background(), sender.ID, sender.Username, fullName)
		if err != nil {
			return err
		}
		if _, err := b.requests.CreateSick(context.Background(), emp.ID, dateFrom, date); err != nil {
			return err
		}
		if err := b.resetState(sender.ID); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"✅ Больничный оформлен!\nС: %s\nПо: %s\n\nКоманда будет получать уведомление каждый день в 8:00.",
			dateFrom.Format("02.01.2006"), date.Format("02.01.2006"),
		))

	case domain.StepPMAwaitDate:
		// ПМ выбирает дату дэйлика
		payload := map[string]string{"date": dateStr}
		if err := b.setState(c.Sender().ID, domain.StepPMAwaitTime, payload); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"🕐 Дата: %s\nВыбери время дэйлика:",
			date.Format("02.01.2006"),
		), timeKeyboard())
	}

	return c.Respond()
}

// handleTimeCallback обрабатывает нажатие кнопки с временем.
func (b *Bot) handleTimeCallback(c tele.Context, timeStr string) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	if state.Step != domain.StepPMAwaitTime {
		return c.Respond()
	}

	_ = c.Edit(c.Message().Text)

	payload := map[string]string{
		"date": state.Payload["date"],
		"time": timeStr,
	}
	if err := b.setState(c.Sender().ID, domain.StepPMAwaitMode, payload); err != nil {
		return err
	}
	return c.Send(fmt.Sprintf("Время: %s\nВыбери формат дэйлика:", timeStr), dailyModeKeyboard())
}

func (b *Bot) handleText(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	switch state.Step {
	case domain.StepPMAwaitLocation:
		return b.handleDailyLocation(c, state)
	default:
		return c.Send("Используй /request чтобы подать заявку, или /daily если ты ПМ.")
	}
}

func (b *Bot) handleMyID(c tele.Context) error {
	return c.Send(fmt.Sprintf("Твой Telegram ID: `%d`", c.Sender().ID), &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})
}

func (b *Bot) handleAddedToGroup(c tele.Context) error {
	chatID := c.Chat().ID
	chatTitle := c.Chat().Title
	log.Printf("Бот добавлен в группу: %q, chat_id: %d", chatTitle, chatID)
	return nil
}
