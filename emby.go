package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type EmbyClient struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

func NewEmbyClient(serverURL, apiKey string) *EmbyClient {
	return &EmbyClient{
		serverURL: NormalizeEmbyURL(serverURL),
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

// NormalizeEmbyURL приводит URL к стандартному виду
func NormalizeEmbyURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	return strings.TrimRight(rawURL, "/")
}

// CheckServer проверяет сетевую доступность сервера Emby
func (c *EmbyClient) CheckServer() error {
	if c.serverURL == "" {
		return fmt.Errorf("URL сервера Emby не указан")
	}

	checkURL := fmt.Sprintf("%s/emby/System/Info/Public", c.serverURL)
	resp, err := c.httpClient.Get(checkURL)
	if err != nil {
		// Компактное сообщение об ошибке без портянки context deadline / net.OpError
		return fmt.Errorf("сервер Emby недоступен по адресу %s", c.serverURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер Emby вернул код статуса %d", resp.StatusCode)
	}

	return nil
}

// CheckToken проверяет валидность API ключа
func (c *EmbyClient) CheckToken() error {
	if err := c.CheckServer(); err != nil {
		return err
	}

	if c.apiKey == "" {
		return fmt.Errorf("API ключ Emby не указан")
	}

	checkURL := fmt.Sprintf("%s/emby/System/Info?api_key=%s", c.serverURL, url.QueryEscape(c.apiKey))
	resp, err := c.httpClient.Get(checkURL)
	if err != nil {
		return fmt.Errorf("ошибка при проверке API ключа Emby: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("недействительный API ключ (код %d)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("неожиданный ответ сервера при проверке ключа (код %d)", resp.StatusCode)
	}

	return nil
}

// TriggerLibraryScan отправляет запрос на сканирование медиатеки
func (c *EmbyClient) TriggerLibraryScan() error {
	if c.serverURL == "" || c.apiKey == "" {
		return fmt.Errorf("данные авторизации Emby не настроены")
	}

	scanURL := fmt.Sprintf("%s/emby/Library/Refresh?api_key=%s", c.serverURL, url.QueryEscape(c.apiKey))
	resp, err := c.httpClient.Post(scanURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса на сканирование Emby: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("сервер Emby вернул код %d при запуске сканирования", resp.StatusCode)
	}

	return nil
}

// PromptForEmbyInteractive выполняет пошаговую настройку Emby в терминале
func PromptForEmbyInteractive() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	var serverURL string
	var apiKey string

	// 1. Ввод и проверка URL сервера
	for {
		fmt.Print("Введите URL сервера Emby (например, http://192.168.1.30:8096): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("Введён пустой адрес. Пропускаем настройку Emby.")
			return "", ""
		}

		serverURL = NormalizeEmbyURL(input)
		client := NewEmbyClient(serverURL, "")

		fmt.Print("Проверка подключения к серверу... ")
		if err := client.CheckServer(); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)

			fmt.Print("Попробовать ввести адрес снова? (Y/n): ")
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println("Настройка Emby отменена.")
				return "", ""
			}
		} else {
			fmt.Println("✅ Сервер найден!")
			break
		}
	}

	// 2. Ввод и проверка API ключа
	for {
		fmt.Print("Введите API ключ Emby: ")
		input, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(input)

		if apiKey == "" {
			fmt.Println("Введён пустой ключ. Настройка Emby отменена.")
			return "", ""
		}

		client := NewEmbyClient(serverURL, apiKey)

		fmt.Print("Проверка API ключа... ")
		if err := client.CheckToken(); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)

			fmt.Print("Попробовать ввести ключ снова? (Y/n): ")
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println("Настройка Emby отменена.")
				return "", ""
			}
		} else {
			fmt.Println("✅ API ключ подтверждён!")
			break
		}
	}

	return serverURL, apiKey
}
