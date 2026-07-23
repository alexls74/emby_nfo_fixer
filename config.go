package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	configDirName  = "emby_nfo_fixer"
	configFileName = "config.conf"
	legacyFileName = "emby_nfo_fixer.conf"
)

type Config struct {
	TmdbToken  string
	EmbyURL    string
	EmbyApiKey string
	Language   string // "ru" или "en"
}

// GetConfigPath определяет путь к конфигурационному файлу
func GetConfigPath() string {
	var userDir string

	if runtime.GOOS == "windows" {
		userDir, _ = os.UserConfigDir() // %APPDATA%
	} else {
		// Для macOS и Linux используем ~/.config
		home, err := os.UserHomeDir()
		if err == nil {
			userDir = filepath.Join(home, ".config")
		} else {
			userDir, _ = os.UserConfigDir()
		}
	}

	targetDir := filepath.Join(userDir, configDirName)
	targetPath := filepath.Join(targetDir, configFileName)

	// Старый путь (рядом с исполняемым файлом)
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	legacyPath := filepath.Join(filepath.Dir(execPath), legacyFileName)

	// Если целевого файла НЕТ, но есть СТАРЫЙ — переносим
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		if data, err := os.ReadFile(legacyPath); err == nil {
			if err := os.MkdirAll(targetDir, 0755); err == nil {
				if err := os.WriteFile(targetPath, data, 0644); err == nil {
					_ = os.Remove(legacyPath)
				}
			}
		}
	}

	return targetPath
}

func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

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
	configPath := GetConfigPath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

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

// promptForTMDBToken запрашивает TMDB токен в консоли
func promptForTMDBToken(httpClient *http.Client) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Файл конфигурации не найден. Хотите настроить TMDB API токен сейчас? (y/N): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" && answer != "д" && answer != "да" {
		fmt.Println("Интеграция с TMDB пропущена.")
		return ""
	}

	for {
		fmt.Print("Введите TMDB API токен: ")
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)

		if token == "" {
			fmt.Println("Введён пустой токен. Пропускаем.")
			return ""
		}

		fmt.Print("Проверка токена... ")
		if err := checkTokenValid(httpClient, token); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)

			fmt.Print("Попробовать ввести снова? (Y/n): ")
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println("Интеграция с TMDB пропущена.")
				return ""
			}
		} else {
			fmt.Println("✅ Токен успешно проверен!")
			return token
		}
	}
}

// EnsureConfig загружает существующий конфиг или запускает мастер настройки при его отсутствии
func EnsureConfig() (*Config, error) {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		httpClient := &http.Client{Timeout: 10 * time.Second}

		token := promptForTMDBToken(httpClient)

		var embyURL, embyKey string
		fmt.Print("Хотите настроить автоматическое сканирование Emby? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		embyAns, _ := reader.ReadString('\n')
		embyAns = strings.ToLower(strings.TrimSpace(embyAns))

		if embyAns == "y" || embyAns == "yes" || embyAns == "д" || embyAns == "да" {
			embyURL, embyKey = PromptForEmbyInteractive()
		}

		cfg := &Config{
			TmdbToken:  token,
			EmbyURL:    embyURL,
			EmbyApiKey: embyKey,
			Language:   "ru",
		}

		if saveErr := SaveConfig(cfg); saveErr != nil {
			return cfg, saveErr
		}

		return cfg, nil
	}

	return LoadConfig()
}
