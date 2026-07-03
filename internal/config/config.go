package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	BotToken    string
	DatabaseURL string
	PMIDs       map[int64]struct{}
	AdminID     int64
	mu          sync.RWMutex
	groupChatID int64
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
		return nil, fmt.Errorf("GROUP_CHAT_ID должен быть числом: %w", err)
	}
	cfg.groupChatID = groupChatID

	pmIDsRaw := os.Getenv("PM_IDS")
	cfg.PMIDs = make(map[int64]struct{})
	for _, part := range strings.Split(pmIDsRaw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("некорректный telegram_id в PM_IDS: %q: %w", part, err)
		}
		cfg.PMIDs[id] = struct{}{}
	}

	adminIDRaw := os.Getenv("ADMIN_ID")
	if adminIDRaw == "" {
		return nil, fmt.Errorf("ADMIN_ID не задан")
	}
	adminID, err := strconv.ParseInt(adminIDRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ADMIN_ID должен быть числом: %w", err)
	}
	cfg.AdminID = adminID

	return cfg, nil
}

func (c *Config) IsPM(telegramID int64) bool {
	_, ok := c.PMIDs[telegramID]
	return ok
}

func (c *Config) GetGroupChatID() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.groupChatID
}

func (c *Config) SetGroupChatID(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groupChatID = id
}
