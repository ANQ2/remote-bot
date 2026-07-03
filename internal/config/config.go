package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken    string
	DatabaseURL string
	GroupChatID int64
	PMIDs       map[int64]struct{}
}

func Load() (*Config, error) {
	cfg := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не задан")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL не задан")
	}

	groupChatIDRaw := os.Getenv("GROUP_CHAT_ID")
	if groupChatIDRaw == "" {
		return nil, fmt.Errorf("GROUP_CHAT_ID не задан")
	}
	groupChatID, err := strconv.ParseInt(groupChatIDRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Group_CHAT_ID должен быть числом: %w", err)
	}
	cfg.GroupChatID = groupChatID

	pmIDsRaw := os.Getenv("PM_IDS")
	cfg.PMIDs = make(map[int64]struct{})
	for _, part := range strings.Split(pmIDsRaw, ",") {
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("некорректный telegram_id в PM_IDS: %q: %w", part, err)
		}
		cfg.PMIDs[id] = struct{}{}
	}

	return cfg, nil
}

func (c *Config) IsPM(telegramID int64) bool {
	_, ok := c.PMIDs[telegramID]
	return ok
}
