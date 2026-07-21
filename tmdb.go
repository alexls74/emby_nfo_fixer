package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const configFileName = "emby_nfo_fixer.conf"

type TMDBClient struct {
	token      string
	httpClient *http.Client
	enabled    bool
}

type tmdbMovieResponse struct {
	ReleaseDate string `json:"release_date"`
}

// checkTokenValid делает тестовый запрос к TMDB
func checkTokenValid(client *http.Client, token string) error {
	req, err := http.NewRequest("GET", "https://api.themoviedb.org/3/movie/550", nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("сетевая ошибка при проверке: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("недействительный токен (401 Unauthorized)")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API вернуло статус %d", resp.StatusCode)
	}

	return nil
}

// promptForToken запрашивает токен у пользователя в терминале
func promptForToken(httpClient *http.Client) string {
	reader := bufio.NewReader(os.Stdin)

	// По умолчанию N (при нажатии Enter — отказ)
	fmt.Print("Файл конфигурации не найден. Хотите настроить TMDB API токен сейчас? (y/N): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" && answer != "д" && answer != "да" {
		fmt.Println("Интеграция с TMDB пропущена. Создан пустой конфигурационный файл.")
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

			// По умолчанию Y (при нажатии Enter — повторить попытку)
			fmt.Print("Попробовать ввести снова? (Y/n): ")
			retry, _ := reader.ReadString('\n')
			retry = strings.ToLower(strings.TrimSpace(retry))

			// Если пользователь явно ввёл 'n', 'no', 'н' или 'нет' — выходим
			if retry == "n" || retry == "no" || retry == "н" || retry == "нет" {
				fmt.Println("Интеграция с TMDB пропущена. Создан пустой конфигурационный файл.")
				return ""
			}
		} else {
			fmt.Println("✅ Токен успешно проверен!")
			return token
		}
	}
}

// getOrInitToken читает токен из конфига рядом с бинарником.
// Если файла нет — запрашивает интерактивно или создает шаблон.
func getOrInitToken(httpClient *http.Client) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	configPath := filepath.Join(execDir, configFileName)

	// Если файла нет — интерактивно запрашиваем и создаем
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		token := promptForToken(httpClient)

		content := fmt.Sprintf("TMDB_TOKEN = %s\n", token)
		_ = os.WriteFile(configPath, []byte(content), 0644)

		if token == "" {
			return "", fmt.Errorf("токен не задан при создании %s", configFileName)
		}
		return token, nil
	}

	// Читаем файл, если он уже существовал
	file, err := os.Open(configPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения %s: %w", configFileName, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if strings.HasPrefix(line, "TMDB_TOKEN") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	// Проверка ошибки чтения сканнера после завершения цикла
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ошибка чтения строк %s: %w", configFileName, err)
	}

	return "", nil
}

func NewTMDBClient() (*TMDBClient, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	token, err := getOrInitToken(httpClient)

	client := &TMDBClient{
		token:      token,
		enabled:    token != "",
		httpClient: httpClient,
	}

	if token == "" {
		if err != nil {
			return client, fmt.Errorf("отсутствует токен в %s (%w)", configFileName, err)
		}
		return client, fmt.Errorf("токен TMDB не задан в файле %s", configFileName)
	}

	return client, nil
}

func (c *TMDBClient) IsEnabled() bool {
	return c != nil && c.enabled
}

func (c *TMDBClient) CheckAvailability() error {
	if !c.IsEnabled() {
		return nil
	}
	return checkTokenValid(c.httpClient, c.token)
}

func (c *TMDBClient) GetReleaseDate(tmdbID string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("модуль TMDB отключен")
	}

	url := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s", tmdbID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+c.token)
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса к TMDB API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("фильм с TMDB ID %s не найден (404 Not Found)", tmdbID)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TMDB API вернуло статус %d для ID %s", resp.StatusCode, tmdbID)
	}

	var movieData tmdbMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&movieData); err != nil {
		return "", fmt.Errorf("ошибка декодирования ответа TMDB: %w", err)
	}

	if movieData.ReleaseDate == "" {
		return "", fmt.Errorf("у фильма с TMDB ID %s отсутствует дата релиза в ответе API", tmdbID)
	}

	return movieData.ReleaseDate, nil
}
