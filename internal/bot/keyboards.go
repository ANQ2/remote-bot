package bot

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

var (
	btnRemote      = tele.Btn{Unique: "type_remote", Text: "🏠 Удалёнка"}
	btnSick        = tele.Btn{Unique: "type_sick", Text: "🤒 Больничный"}
	btnDailyShort  = tele.Btn{Unique: "type_daily", Text: "📅 Дэйлик"}
	btnOnline      = tele.Btn{Unique: "mode_online", Text: "💻 Онлайн"}
	btnOffline     = tele.Btn{Unique: "mode_offline", Text: "🏢 Офлайн"}
	btnConfirmSend = tele.Btn{Unique: "confirm_send", Text: "✅ Отправить"}
	btnEditTime    = tele.Btn{Unique: "edit_time", Text: "✏️ Изменить время"}
	btnEditLoc     = tele.Btn{Unique: "edit_location", Text: "✏️ Изменить адрес/ссылку"}
	btnUseLastLoc  = tele.Btn{Unique: "use_last_loc", Text: "♻️ Использовать прошлый"}
)

func requestTypeKeyboard(isPM bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	if isPM {
		kb.Inline(
			kb.Row(btnRemote, btnSick),
			kb.Row(btnDailyShort),
		)
	} else {
		kb.Inline(kb.Row(btnRemote, btnSick))
	}
	return kb
}

func dailyModeKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(btnOnline, btnOffline))
	return kb
}

func confirmKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(btnConfirmSend),
		kb.Row(btnEditTime, btnEditLoc),
	)
	return kb
}

func locationKeyboard(hasLast bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	if hasLast {
		kb.Inline(kb.Row(btnUseLastLoc))
	}
	return kb
}

func weekDatesKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	now := time.Now()

	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var rows []tele.Row
	for week := 0; week < 2; week++ {
		var row []tele.Btn
		for day := 0; day < 5; day++ {
			date := monday.AddDate(0, 0, week*7+day)
			if date.Before(today) {
				continue
			}
			label := date.Format("02.01")
			unique := fmt.Sprintf("date_%s", date.Format("2006-01-02"))
			row = append(row, tele.Btn{Unique: unique, Text: label})
		}
		if len(row) > 0 {
			rows = append(rows, kb.Row(row...))
		}
	}
	kb.Inline(rows...)
	return kb
}

func timeKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	var row []tele.Btn

	for h := 9; h <= 18; h++ {
		for _, m := range []int{0, 30} {
			if h == 18 && m == 30 {
				break
			}
			timeStr := fmt.Sprintf("%02d:%02d", h, m)
			unique := fmt.Sprintf("time_%s", strings.ReplaceAll(timeStr, ":", "_"))
			row = append(row, tele.Btn{Unique: unique, Text: timeStr})
			if len(row) == 4 {
				rows = append(rows, kb.Row(row...))
				row = nil
			}
		}
	}
	if len(row) > 0 {
		rows = append(rows, kb.Row(row...))
	}
	kb.Inline(rows...)
	return kb
}
