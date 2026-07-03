package bot

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
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
		payload := map[string]string{
			"type": state.Payload["type"],
			"date": dateStr,
		}
		if err := b.setState(c.Sender().ID, domain.StepConfirmDate, payload); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"🏠 Удалёнка\n📅 Дата: %s\n\nВсё верно?",
			date.Format("02.01.2006"),
		), confirmDateKeyboard())

	case domain.StepAwaitSickFrom:
		payload := map[string]string{"date_from": dateStr}
		if err := b.setState(c.Sender().ID, domain.StepAwaitSickTo, payload); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"📅 Дата начала: %s\nТеперь выбери дату окончания больничного:",
			date.Format("02.01.2006"),
		), weekDatesKeyboard())

	case domain.StepAwaitSickTo:
		dateFrom, err := time.Parse("2006-01-02", state.Payload["date_from"])
		if err != nil {
			_ = b.resetState(c.Sender().ID)
			return c.Send("⚠️ Ошибка, начни заново: /request")
		}
		if date.Before(dateFrom) {
			return c.Send("⚠️ Дата окончания не может быть раньше даты начала. Выбери снова:", weekDatesKeyboard())
		}
		payload := map[string]string{
			"date_from": state.Payload["date_from"],
			"date_to":   dateStr,
		}
		if err := b.setState(c.Sender().ID, domain.StepConfirmSick, payload); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"🤒 Больничный\n📅 С: %s\n📅 По: %s\n\nВсё верно?",
			dateFrom.Format("02.01.2006"), date.Format("02.01.2006"),
		), confirmDateKeyboard())

	case domain.StepPMAwaitDate:
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

func (b *Bot) handleBtnConfirmDate(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}

	_ = c.Edit(c.Message().Text)
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}

	switch state.Step {
	case domain.StepConfirmDate:
		date, err := time.Parse("2006-01-02", state.Payload["date"])
		if err != nil {
			_ = b.resetState(sender.ID)
			return c.Send("⚠️ Ошибка, начни заново: /request")
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
		_ = b.notifier.NotifyGroupRemoteRequest(domain.RequestWithEmployee{
			Request:            domain.Request{Type: domain.RequestRemote, Date: date},
			EmployeeFullName:   fullName,
			EmployeeTelegramID: sender.ID,
			EmployeeUsername:   sender.Username,
		})
		return c.Send(fmt.Sprintf(
			"✅ Заявка принята!\nТип: удалёнка\nДата: %s\n\nВ 8:00 утра команда также получит общий список.",
			date.Format("02.01.2006"),
		), cancelLastRequestKeyboard())

	case domain.StepConfirmSick:
		dateFrom, _ := time.Parse("2006-01-02", state.Payload["date_from"])
		dateTo, _ := time.Parse("2006-01-02", state.Payload["date_to"])
		emp, err := b.employees.GetOrCreate(context.Background(), sender.ID, sender.Username, fullName)
		if err != nil {
			return err
		}
		if _, err := b.requests.CreateSick(context.Background(), emp.ID, dateFrom, dateTo); err != nil {
			return err
		}
		if err := b.resetState(sender.ID); err != nil {
			return err
		}
		return c.Send(fmt.Sprintf(
			"✅ Больничный оформлен!\nС: %s\nПо: %s\n\nКоманда будет получать уведомление каждый день в 8:00.",
			dateFrom.Format("02.01.2006"), dateTo.Format("02.01.2006"),
		), cancelLastRequestKeyboard())
	}
	return c.Respond()
}

func (b *Bot) handleBtnChangeDate(c tele.Context) error {
	state, err := b.getState(c)
	if err != nil {
		return err
	}
	_ = c.Edit(c.Message().Text)

	switch state.Step {
	case domain.StepConfirmDate:
		payload := map[string]string{"type": state.Payload["type"]}
		if err := b.setState(c.Sender().ID, domain.StepAwaitDate, payload); err != nil {
			return err
		}
		return c.Send("📅 Выбери новую дату удалёнки:", weekDatesKeyboard())

	case domain.StepConfirmSick:
		if err := b.setState(c.Sender().ID, domain.StepAwaitSickFrom, nil); err != nil {
			return err
		}
		return c.Send("📅 Выбери новую дату начала больничного:", weekDatesKeyboard())
	}
	return c.Respond()
}

func (b *Bot) handleBtnCancelRequest(c tele.Context) error {
	_ = c.Edit(c.Message().Text)
	if err := b.resetState(c.Sender().ID); err != nil {
		return err
	}
	return c.Send("❌ Заявка отменена.")
}

func (b *Bot) handleBtnCancelLastRequest(c tele.Context) error {
	_ = c.Edit(c.Message().Text)
	sender := c.Sender()
	fullName := sender.FirstName
	if sender.LastName != "" {
		fullName += " " + sender.LastName
	}
	emp, err := b.employees.GetOrCreate(context.Background(), sender.ID, sender.Username, fullName)
	if err != nil {
		return err
	}
	req, err := b.requests.DeleteLastByEmployee(context.Background(), emp.ID)
	if err != nil {
		return c.Send("❌ Не удалось отменить заявку.")
	}
	_ = b.notifier.NotifyGroupCancelRequest(fullName, sender.Username, req.Type, req.Date)
	return c.Send("✅ Заявка отменена. Команда получила уведомление.")
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
		return nil
	}
}

func (b *Bot) handleMyID(c tele.Context) error {
	return c.Send(fmt.Sprintf("Твой Telegram ID: `%d`", c.Sender().ID), &tele.SendOptions{
		ParseMode: tele.ModeMarkdown,
	})
}

func (b *Bot) handleSetChat(c tele.Context) error {
	if c.Sender().ID != b.cfg.AdminID {
		return c.Send("⛔ Нет доступа.")
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Использование: /setchat -1001234567890\n\nИли перешли любое сообщение из группы — я покажу chat_id.")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send("⚠️ Неверный формат. Пример: /setchat -1001234567890")
	}
	b.cfg.SetGroupChatID(id)
	return c.Send(fmt.Sprintf(
		"✅ GROUP_CHAT_ID обновлён: %d\n\n⚠️ После перезапуска нужно обновить .env:\nGROUP_CHAT_ID=%d",
		id, id,
	))
}

func (b *Bot) handleForward(c tele.Context) error {
	if c.Sender().ID != b.cfg.AdminID {
		return nil
	}
	chatID := c.Message().Chat.ID
	return c.Send(fmt.Sprintf(
		"Chat ID: `%d`\n\nЧтобы установить: /setchat %d",
		chatID, chatID,
	), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

func (b *Bot) handleLogs(c tele.Context) error {
	if c.Sender().ID != b.cfg.AdminID {
		return c.Send("⛔ Нет доступа.")
	}
	out, err := exec.Command("journalctl", "-u", "teampulse", "-n", "30", "--no-pager").Output()
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Ошибка получения логов: %v", err))
	}
	text := string(out)
	if len(text) > 4000 {
		text = text[len(text)-4000:]
	}
	return c.Send("📋 Последние логи:\n\n" + text)
}

func (b *Bot) handleAddedToGroup(c tele.Context) error {
	chatID := c.Chat().ID
	chatTitle := c.Chat().Title
	log.Printf("Бот добавлен в группу: %q, chat_id: %d", chatTitle, chatID)
	return nil
}
