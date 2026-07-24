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

// detectSystemLanguage определяет системный язык ОС
func detectSystemLanguage() string {
	var langEnv string

	if runtime.GOOS == "windows" {
		langEnv = os.Getenv("LANG")
		if langEnv == "" {
			langEnv = os.Getenv("LC_ALL")
		}
	} else {
		langEnv = os.Getenv("LC_ALL")
		if langEnv == "" {
			langEnv = os.Getenv("LANG")
		}
		if langEnv == "" {
			langEnv = os.Getenv("LC_MESSAGES")
		}
	}

	langEnv = strings.ToLower(langEnv)

	if strings.HasPrefix(langEnv, "en") {
		return "en"
	}

	return "ru"
}

// GetConfigPath определяет путь к конфигурационному файлу
func GetConfigPath() string {
	var userDir string

	if runtime.GOOS == "windows" {
		userDir, _ = os.UserConfigDir()
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			userDir = filepath.Join(home, ".config")
		} else {
			userDir, _ = os.UserConfigDir()
		}
	}

	targetDir := filepath.Join(userDir, configDirName)
	targetPath := filepath.Join(targetDir, configFileName)

	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	legacyPath := filepath.Join(filepath.Dir(execPath), legacyFileName)

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

	cfg := &Config{}

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
				valLower := strings.ToLower(val)
				if valLower == "en" || valLower == "english" {
					cfg.Language = "en"
				} else if valLower == "ru" || valLower == "russian" {
					cfg.Language = "ru"
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
		cfg.Language = detectSystemLanguage()
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

func promptForLanguage() string {
	reader := bufio.NewReader(os.Stdin)

	defaultLang := detectSystemLanguage()
	defaultChoice := "1"
	if defaultLang == "en" {
		defaultChoice = "2"
	}

	fmt.Println("Выберите язык интерфейса / Select interface language:")
	fmt.Println("  1. Русский (ru)")
	fmt.Println("  2. English (en)")
	fmt.Printf("(по умолчанию %s, default %s): ", defaultChoice, defaultChoice)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultLang
	}

	if input == "2" || strings.ToLower(input) == "en" || strings.ToLower(input) == "english" {
		return "en"
	}

	return "ru"
}

func promptForTMDBToken(httpClient *http.Client) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(T("ask_tmdb_token"))
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" && answer != "д" && answer != "да" {
		fmt.Println(T("skip_tmdb"))
		return ""
	}

	for {
		fmt.Print(T("enter_tmdb_token"))
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)

		if token == "" {
			fmt.Println(T("empty_token_skip"))
			return ""
		}

		fmt.Print(T("checking_token"))
		if err := checkTokenValid(httpClient, token); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)

			fmt.Print(T("token_retry_prompt"))
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println(T("skip_tmdb"))
				return ""
			}
		} else {
			fmt.Println(T("token_success"))
			return token
		}
	}
}

// EnsureConfig загружает существующий конфиг или запускает мастер настройки при его отсутствии
func EnsureConfig() (*Config, error) {
	configPath := GetConfigPath()

	// 1. Мастер первичной настройки, если конфига вообще нет
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		lang := promptForLanguage()
		SetLanguage(lang)
		fmt.Println()

		fmt.Println(T("config_not_found"))
		fmt.Println()

		httpClient := &http.Client{Timeout: 10 * time.Second}

		token := promptForTMDBToken(httpClient)

		var embyURL, embyKey string
		fmt.Print(T("ask_emby_setup"))
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
			Language:   lang,
		}

		if saveErr := SaveConfig(cfg); saveErr != nil {
			return cfg, saveErr
		}

		return cfg, nil
	}

	// 2. Загружаем существующий конфиг
	cfg, err := LoadConfig()
	if err != nil {
		return cfg, err
	}

	// 3. Если конфиг существует, но ключа LANGUAGE в нём не было — спрашиваем и дозаписываем
	if cfg.Language == "" {
		cfg.Language = promptForLanguage()
		_ = SaveConfig(cfg) // сохраняем выбор в конфиг, чтобы не спрашивать снова
		fmt.Println()
	}

	return cfg, nil
}
