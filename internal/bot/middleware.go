package bot

import (
	tele "gopkg.in/telebot.v3"

	"remote-bot/internal/config"
)

func onlyPM(cfg *config.Config) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if !cfg.IsPM(c.Sender().ID) {
				return c.Send("⛔ Эта команда доступна только менеджеру проекта.")
			}
			return next(c)
		}
	}
}

func onlyPrivate() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Chat().Type != tele.ChatPrivate {
				return nil
			}
			return next(c)
		}
	}
}
