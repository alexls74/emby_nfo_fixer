package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type UniqueID struct {
	Type    string `xml:"type,attr"`
	Default string `xml:"default,attr,omitempty"`
	Value   string `xml:",chardata"`
}

type Movie struct {
	Path              string     `xml:"-"`
	RawContent        string     `xml:"-"`
	Lines             []string   `xml:"-"`
	HasPremiered      bool       `xml:"-"`
	TmdbID            string     `xml:"-"`
	PremieredDate     string     `xml:"-"`
	NeedsPremieredFix bool       `xml:"-"`
	NeedsFix          bool       `xml:"-"`
	MovedCredits      int        `xml:"-"`
	MovedDirectors    int        `xml:"-"`
	UniqueIDs         []UniqueID `xml:"uniqueid"`
	TmdbIDTag         string     `xml:"tmdbid"`
}

// LoadMovie загружает и первично анализирует NFO файл
func LoadMovie(path string) (*Movie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	content := string(data)
	movie := &Movie{
		Path:       path,
		RawContent: content,
		Lines:      strings.Split(content, "\n"),
	}

	// Чтение структуры через XML парсер для проверки тегов ID и premiered
	var xmlData struct {
		Premiered string     `xml:"premiered"`
		TmdbID    string     `xml:"tmdbid"`
		UniqueIDs []UniqueID `xml:"uniqueid"`
	}

	if err := xml.Unmarshal(data, &xmlData); err == nil {
		if strings.TrimSpace(xmlData.Premiered) != "" {
			movie.HasPremiered = true
		}

		// 1. Поиск TMDB ID в стандартном теге <tmdbid>
		if strings.TrimSpace(xmlData.TmdbID) != "" {
			movie.TmdbID = strings.TrimSpace(xmlData.TmdbID)
		}

		// 2. Если в <tmdbid> пусто, ищем в <uniqueid type="tmdb">
		if movie.TmdbID == "" {
			for _, uid := range xmlData.UniqueIDs {
				if strings.ToLower(uid.Type) == "tmdb" && strings.TrimSpace(uid.Value) != "" {
					movie.TmdbID = strings.TrimSpace(uid.Value)
					break
				}
			}
		}
	} else {
		// Резервный поиск регулярными выражениями, если XML нестрогий
		if regexp.MustCompile(`(?i)<premiered>.*?</premiered>`).MatchString(content) {
			movie.HasPremiered = true
		}

		tmdbMatch := regexp.MustCompile(`(?i)<tmdbid>(\d+)</tmdbid>`).FindStringSubmatch(content)
		if len(tmdbMatch) > 1 {
			movie.TmdbID = tmdbMatch[1]
		} else {
			uidMatch := regexp.MustCompile(`(?i)<uniqueid[^>]*type=["']tmdb["'][^>]*>(\d+)</uniqueid>`).FindStringSubmatch(content)
			if len(uidMatch) > 1 {
				movie.TmdbID = uidMatch[1]
			}
		}
	}

	movie.Analyze()
	return movie, nil
}

// Analyze проверяет порядок тегов <credits> и <director> относительно <actor>
func (m *Movie) Analyze() {
	firstActorLine := -1
	creditsLines := []int{}
	directorLines := []int{}

	for i, line := range m.Lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<actor>") || strings.HasPrefix(trimmed, "<actor ") {
			if firstActorLine == -1 {
				firstActorLine = i
			}
		} else if strings.HasPrefix(trimmed, "<credits>") || strings.HasPrefix(trimmed, "<credits ") {
			creditsLines = append(creditsLines, i)
		} else if strings.HasPrefix(trimmed, "<director>") || strings.HasPrefix(trimmed, "<director ") {
			directorLines = append(directorLines, i)
		}
	}

	// Если есть актеры, проверяем, стоят ли режиссер и сценаристы ДО актеров
	if firstActorLine != -1 {
		for _, lineIdx := range creditsLines {
			if lineIdx < firstActorLine {
				m.MovedCredits++
			}
		}
		for _, lineIdx := range directorLines {
			if lineIdx < firstActorLine {
				m.MovedDirectors++
			}
		}
	}

	if m.MovedCredits > 0 || m.MovedDirectors > 0 {
		m.NeedsFix = true
	}
}

// Fix выполняет перемещение тегов и добавление даты премьеры
func (m *Movie) Fix() {
	var newLines []string
	var extractedCredits []string
	var extractedDirectors []string

	inCredits := false
	inDirector := false

	// Извлекаем <credits> и <director>
	for _, line := range m.Lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "<credits>") || strings.HasPrefix(trimmed, "<credits ") {
			inCredits = true
			extractedCredits = append(extractedCredits, line)
			if strings.Contains(trimmed, "</credits>") {
				inCredits = false
			}
			continue
		}
		if inCredits {
			extractedCredits = append(extractedCredits, line)
			if strings.Contains(trimmed, "</credits>") {
				inCredits = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "<director>") || strings.HasPrefix(trimmed, "<director ") {
			inDirector = true
			extractedDirectors = append(extractedDirectors, line)
			if strings.Contains(trimmed, "</director>") {
				inDirector = false
			}
			continue
		}
		if inDirector {
			extractedDirectors = append(extractedDirectors, line)
			if strings.Contains(trimmed, "</director>") {
				inDirector = false
			}
			continue
		}

		newLines = append(newLines, line)
	}

	// Вставляем извлеченные теги строго ПОСЛЕ последнего блока <actor>
	var finalLines []string
	lastActorEndIndex := -1

	for i, line := range newLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "</actor>") {
			lastActorEndIndex = i
		}
	}

	if lastActorEndIndex != -1 {
		for i, line := range newLines {
			finalLines = append(finalLines, line)
			if i == lastActorEndIndex {
				finalLines = append(finalLines, extractedCredits...)
				finalLines = append(finalLines, extractedDirectors...)
			}
		}
	} else {
		// Если актеров нет, вставляем перед закрывающим тегом </movie>
		for _, line := range newLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "</movie>") {
				finalLines = append(finalLines, extractedCredits...)
				finalLines = append(finalLines, extractedDirectors...)
			}
			finalLines = append(finalLines, line)
		}
	}

	// Добавление тега <premiered>, если его не было
	if m.NeedsPremieredFix && m.PremieredDate != "" {
		var premieredLines []string
		inserted := false

		// Определяем отступ для красивого форматирования XML
		indent := "    "
		for _, line := range finalLines {
			if strings.Contains(line, "<year>") || strings.Contains(line, "<title>") {
				indent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				break
			}
		}

		premieredTag := fmt.Sprintf("%s<premiered>%s</premiered>", indent, m.PremieredDate)

		for _, line := range finalLines {
			trimmed := strings.TrimSpace(line)
			if !inserted && (strings.HasPrefix(trimmed, "<year>") || strings.HasPrefix(trimmed, "<plot>")) {
				premieredLines = append(premieredLines, premieredTag)
				inserted = true
			}
			premieredLines = append(premieredLines, line)
		}

		if !inserted {
			// Если не нашли <year> или <plot>, добавляем после <movie>
			var fallbackLines []string
			for _, line := range premieredLines {
				fallbackLines = append(fallbackLines, line)
				if !inserted && strings.Contains(line, "<movie>") {
					fallbackLines = append(fallbackLines, premieredTag)
					inserted = true
				}
			}
			premieredLines = fallbackLines
		}

		finalLines = premieredLines
	}

	m.Lines = finalLines
	m.RawContent = strings.Join(finalLines, "\n")
}

// Save сохраняет изменения обратно в файл
func (m *Movie) Save() error {
	return os.WriteFile(m.Path, []byte(m.RawContent), 0644)
}
