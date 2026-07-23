package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configFileName = "emby_nfo_fixer.conf"

type Config struct {
	TmdbToken  string
	EmbyURL    string
	EmbyApiKey string
	Language   string // "ru" или "en"
}

func getConfigPath() string {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, configFileName)
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

	// По умолчанию ставим русский язык
	cfg := &Config{
		Language: "ru",
	}

	file, err := os.Open(configPath)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "TMDB_TOKEN":
				cfg.TmdbToken = val
			case "EMBY_URL":
				cfg.EmbyURL = val
			case "EMBY_API_KEY":
				cfg.EmbyApiKey = val
			case "LANGUAGE":
				if val == "en" || val == "ru" {
					cfg.Language = val
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return cfg, fmt.Errorf("ошибка чтения строк %s: %w", configFileName, err)
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	configPath := getConfigPath()

	if cfg.Language == "" {
		cfg.Language = "ru"
	}

	content := fmt.Sprintf(
		"TMDB_TOKEN = %s\nEMBY_URL = %s\nEMBY_API_KEY = %s\nLANGUAGE = %s\n",
		cfg.TmdbToken,
		cfg.EmbyURL,
		cfg.EmbyApiKey,
		cfg.Language,
	)

	return os.WriteFile(configPath, []byte(content), 0644)
}
