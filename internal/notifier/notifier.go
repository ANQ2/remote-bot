package notifier

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"remote-bot/internal/config"
	"remote-bot/internal/domain"
)

type Notifier struct {
	bot *tele.Bot
	cfg *config.Config
}

func New(bot *tele.Bot, cfg *config.Config) *Notifier {
	return &Notifier{bot: bot, cfg: cfg}
}

func (n *Notifier) NotifyGroupRequests(requests []domain.RequestWithEmployee) error {
	if len(requests) == 0 {
		return nil
	}

	var remotes []domain.RequestWithEmployee
	var sicks []domain.RequestWithEmployee
	for _, r := range requests {
		if r.Type == domain.RequestRemote {
			remotes = append(remotes, r)
		} else {
			sicks = append(sicks, r)
		}
	}

	var sb strings.Builder

	if len(remotes) > 0 {
		sb.WriteString("🏠 Заявки на удалёнку на сегодня:\n")
		for i, r := range remotes {
			name := formatName(r)
			sb.WriteString(fmt.Sprintf("%d. %s — %s\n", i+1, name, r.Date.Format("02.01.2006")))
		}
	}

	if len(sicks) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("🤒 На больничном:\n")
		for i, r := range sicks {
			name := formatName(r)
			var dateStr string
			if r.DateFrom != nil && r.DateTo != nil {
				dateStr = fmt.Sprintf("%s до %s", r.DateFrom.Format("02.01.2006"), r.DateTo.Format("02.01.2006"))
			} else {
				dateStr = r.Date.Format("02.01.2006")
			}
			sb.WriteString(fmt.Sprintf("%d. %s — %s\n", i+1, name, dateStr))
		}
	}

	return n.sendToGroup(strings.TrimSpace(sb.String()))
}

func (n *Notifier) NotifyGroupRemoteRequest(emp domain.RequestWithEmployee) error {
	name := formatName(emp)
	text := fmt.Sprintf("🏠 %s — %s просится на удалёнку", name, emp.Date.Format("02.01.2006"))
	return n.sendToGroup(text)
}

func (n *Notifier) NotifyGroupDaily(d domain.Daily) error {
	var text string
	if d.Mode == domain.DailyOnline {
		text = fmt.Sprintf("📅 Дэйлик онлайн 💻\nДата: %s\nВремя: %s\nСсылка: %s",
			d.Date.Format("02.01.2006"), d.Time, d.Location)
	} else {
		text = fmt.Sprintf("📅 Дэйлик офлайн 🏢\nДата: %s\nВремя: %s\nАдрес: %s",
			d.Date.Format("02.01.2006"), d.Time, d.Location)
	}
	return n.sendToGroup(text)
}

func (n *Notifier) NotifyGroupCancelRequest(fullName, username string, reqType domain.RequestType, date time.Time) error {
	var name string
	if username != "" {
		name = fmt.Sprintf("%s (@%s)", fullName, username)
	} else {
		name = fullName
	}
	var typeLabel string
	if reqType == domain.RequestRemote {
		typeLabel = "удалёнку"
	} else {
		typeLabel = "больничный"
	}
	text := fmt.Sprintf("🚫 %s отменил заявку на %s — %s", name, typeLabel, date.Format("02.01.2006"))
	return n.sendToGroup(text)
}

func (n *Notifier) NotifyGroupCancelDaily(fullName, username string, d domain.Daily) error {
	var name string
	if username != "" {
		name = fmt.Sprintf("%s (@%s)", fullName, username)
	} else {
		name = fullName
	}
	text := fmt.Sprintf("🚫 %s отменил дэйлик %s в %s", name, d.Date.Format("02.01.2006"), d.Time)
	return n.sendToGroup(text)
}

func (n *Notifier) SendToUser(telegramID int64, text string) error {
	chat := &tele.Chat{ID: telegramID}
	_, err := n.bot.Send(chat, text)
	if err != nil {
		return fmt.Errorf("send to user %d: %w", telegramID, err)
	}
	return nil
}

func (n *Notifier) sendToGroup(text string) error {
	chat := &tele.Chat{ID: n.cfg.GetGroupChatID()}
	_, err := n.bot.Send(chat, text)
	if err != nil {
		return fmt.Errorf("send to group: %w", err)
	}
	return nil
}

func formatName(r domain.RequestWithEmployee) string {
	if r.EmployeeUsername != "" {
		return fmt.Sprintf("%s (@%s)", r.EmployeeFullName, r.EmployeeUsername)
	}
	return r.EmployeeFullName
}
