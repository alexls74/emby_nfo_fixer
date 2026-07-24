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
		return fmt.Errorf("%s", T("err_emby_no_url"))
	}

	checkURL := fmt.Sprintf("%s/emby/System/Info/Public", c.serverURL)
	resp, err := c.httpClient.Get(checkURL)
	if err != nil {
		return fmt.Errorf(T("err_emby_unavail"), c.serverURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(T("err_emby_status"), resp.StatusCode)
	}

	return nil
}

// CheckToken проверяет валидность API ключа
func (c *EmbyClient) CheckToken() error {
	if err := c.CheckServer(); err != nil {
		return err
	}

	if c.apiKey == "" {
		return fmt.Errorf("%s", T("err_emby_no_key"))
	}

	checkURL := fmt.Sprintf("%s/emby/System/Info?api_key=%s", c.serverURL, url.QueryEscape(c.apiKey))
	resp, err := c.httpClient.Get(checkURL)
	if err != nil {
		return fmt.Errorf(T("err_emby_key_check"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf(T("err_emby_invalid_key"), resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(T("err_emby_unexpected"), resp.StatusCode)
	}

	return nil
}

// TriggerLibraryScan отправляет запрос на сканирование медиатеки
func (c *EmbyClient) TriggerLibraryScan() error {
	if c.serverURL == "" || c.apiKey == "" {
		return fmt.Errorf("%s", T("err_emby_scan_auth"))
	}

	scanURL := fmt.Sprintf("%s/emby/Library/Refresh?api_key=%s", c.serverURL, url.QueryEscape(c.apiKey))
	resp, err := c.httpClient.Post(scanURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf(T("err_emby_scan_send"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf(T("err_emby_status"), resp.StatusCode)
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
		fmt.Print(T("enter_emby_url"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println(T("empty_url_skip"))
			return "", ""
		}

		serverURL = NormalizeEmbyURL(input)
		client := NewEmbyClient(serverURL, "")

		fmt.Print(T("checking_server"))
		if err := client.CheckServer(); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)

			fmt.Print(T("url_retry_prompt"))
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println(T("emby_canceled"))
				return "", ""
			}
		} else {
			fmt.Println(T("server_found"))
			break
		}
	}

	// 2. Ввод и проверка API ключа
	for {
		fmt.Print(T("enter_emby_api_key"))
		input, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(input)

		if apiKey == "" {
			fmt.Println(T("empty_key_skip"))
			return "", ""
		}

		client := NewEmbyClient(serverURL, apiKey)

		fmt.Print(T("checking_key"))
		if err := client.CheckToken(); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)

			fmt.Print(T("key_retry_prompt"))
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println(T("emby_canceled"))
				return "", ""
			}
		} else {
			fmt.Println(T("key_confirmed"))
			break
		}
	}

	return serverURL, apiKey
}
