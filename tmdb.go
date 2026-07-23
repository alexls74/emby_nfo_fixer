package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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

// NewTMDBClient создает клиент по переданному токену
func NewTMDBClient(token string) *TMDBClient {
	return &TMDBClient{
		token:   token,
		enabled: token != "",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *TMDBClient) IsEnabled() bool {
	return c != nil && c.enabled
}

func (c *TMDBClient) CheckAvailability() error {
	if !c.IsEnabled() {
		return fmt.Errorf("токен TMDB не задан в файле %s", configFileName)
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
