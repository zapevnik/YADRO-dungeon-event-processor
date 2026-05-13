package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds dungeon configuration loaded from a JSON file.
type Config struct {
	Floors   int    `json:"Floors"`
	Monsters int    `json:"Monsters"`
	OpenAt   string `json:"OpenAt"`
	Duration int    `json:"Duration"`

	OpenAtTime  time.Time
	CloseAtTime time.Time
}

// Load reads and parses a JSON config file.
func LoadCfg(path string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return &cfg, err
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return &cfg, err
	}

	err = cfg.parse()

	return &cfg, err
}

func (c *Config) parse() error {
	t, err := time.Parse("15:04:05", c.OpenAt)
	if err != nil {
		return fmt.Errorf("parsing OpenAt %q: %w", c.OpenAt, err)
	}
	c.OpenAtTime = t
	c.CloseAtTime = t.Add(time.Duration(c.Duration) * time.Hour)
	return nil
}