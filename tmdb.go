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
		return fmt.Errorf(T("err_tmdb_req_create"), err)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(T("err_tmdb_network"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%s", T("err_tmdb_401"))
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
		return fmt.Errorf(T("err_tmdb_no_token"), configFileName)
	}
	return checkTokenValid(c.httpClient, c.token)
}

func (c *TMDBClient) GetReleaseDate(tmdbID string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("%s", T("err_tmdb_disabled"))
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
		return "", fmt.Errorf(T("err_tmdb_not_found"), tmdbID)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(T("err_tmdb_api_status"), resp.StatusCode, tmdbID)
	}

	var movieData tmdbMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&movieData); err != nil {
		return "", fmt.Errorf(T("err_tmdb_decode"), err)
	}

	if movieData.ReleaseDate == "" {
		return "", fmt.Errorf(T("err_tmdb_no_release"), tmdbID)
	}

	return movieData.ReleaseDate, nil
}
