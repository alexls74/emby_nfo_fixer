package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var tmdbIDRegexp = regexp.MustCompile(`<tmdbid>([0-9]+)</tmdbid>`)

type Movie struct {
	File string

	Content string

	Credits   []string
	Directors []string
	Actors    []string

	MovedCredits   int
	MovedDirectors int

	LastMovedCredits   int
	LastMovedDirectors int

	// Поля для работы с TMDB и тегом <premiered>
	TmdbID            string
	HasPremiered      bool
	NeedsPremieredFix bool
	PremieredDate     string

	NeedsFix bool
}

// LoadMovie загружает и анализирует NFO файл фильма
func LoadMovie(filename string) (*Movie, error) {

	data, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf(
			"ошибка чтения файла: %w",
			err,
		)
	}

	content := string(data)

	movie := &Movie{
		File:    filename,
		Content: content,
	}

	movie.analyze()

	return movie, nil
}

func (m *Movie) analyze() {

	lines := strings.Split(
		m.Content,
		"\n",
	)

	var (
		actorIndexes   []int
		specialIndexes []int
	)

	m.MovedCredits = 0
	m.MovedDirectors = 0
	m.NeedsFix = false
	m.HasPremiered = false
	m.TmdbID = ""

	m.Credits = nil
	m.Directors = nil
	m.Actors = nil

	// Поиск TMDB ID в файле
	if matches := tmdbIDRegexp.FindStringSubmatch(m.Content); len(matches) > 1 {
		m.TmdbID = matches[1]
	}

	for i, line := range lines {

		trim := strings.TrimSpace(line)

		if strings.HasPrefix(trim, "<premiered>") {
			m.HasPremiered = true
		}

		switch {

		case strings.HasPrefix(trim, "<actor>"):

			actorIndexes = append(
				actorIndexes,
				i,
			)

			m.Actors = append(
				m.Actors,
				line,
			)

		case strings.HasPrefix(trim, "<credits>"):

			m.Credits = append(
				m.Credits,
				line,
			)

			specialIndexes = append(
				specialIndexes,
				i,
			)

		case strings.HasPrefix(trim, "<director>"):

			m.Directors = append(
				m.Directors,
				line,
			)

			specialIndexes = append(
				specialIndexes,
				i,
			)
		}
	}

	if len(actorIndexes) > 0 {
		lastActor := actorIndexes[len(actorIndexes)-1]

		for _, index := range specialIndexes {

			if index < lastActor {

				m.NeedsFix = true

				trim := strings.TrimSpace(
					lines[index],
				)

				switch {

				case strings.HasPrefix(trim, "<credits>"):
					m.MovedCredits++

				case strings.HasPrefix(trim, "<director>"):
					m.MovedDirectors++
				}
			}
		}
	}
}

func (m *Movie) Fix() {

	m.LastMovedCredits = m.MovedCredits
	m.LastMovedDirectors = m.MovedDirectors

	lines := strings.Split(
		m.Content,
		"\n",
	)

	// 1. Исправление структуры
	if m.MovedCredits > 0 || m.MovedDirectors > 0 {
		var result []string

		for _, line := range lines {

			trim := strings.TrimSpace(line)

			if strings.HasPrefix(trim, "<credits>") ||
				strings.HasPrefix(trim, "<director>") {

				continue
			}

			result = append(
				result,
				line,
			)
		}

		lastActor := -1

		for i, line := range result {

			if strings.HasPrefix(
				strings.TrimSpace(line),
				"</actor>",
			) {

				lastActor = i
			}
		}

		if lastActor != -1 {
			insert := []string{}

			insert = append(
				insert,
				m.Credits...,
			)

			insert = append(
				insert,
				m.Directors...,
			)

			final := make(
				[]string,
				0,
				len(result)+len(insert),
			)

			final = append(
				final,
				result[:lastActor+1]...,
			)

			final = append(
				final,
				insert...,
			)

			final = append(
				final,
				result[lastActor+1:]...,
			)

			lines = final
		}
	}

	// 2. Вставка <premiered> сразу после <year> с сохранением отступа
	if m.NeedsPremieredFix && m.PremieredDate != "" {
		var finalLines []string
		premieredInserted := false

		for _, line := range lines {
			finalLines = append(finalLines, line)

			trim := strings.TrimSpace(line)
			if !premieredInserted && strings.HasPrefix(trim, "<year>") {
				// Вычисляем оригинальный отступ
				indentLen := len(line) - len(strings.TrimLeft(line, " \t"))
				indent := line[:indentLen]

				premieredLine := fmt.Sprintf("%s<premiered>%s</premiered>", indent, m.PremieredDate)
				finalLines = append(finalLines, premieredLine)
				premieredInserted = true
			}
		}

		lines = finalLines
	}

	m.Content = strings.Join(
		lines,
		"\n",
	)

	m.analyze()
}

func (m *Movie) Save() error {

	err := os.WriteFile(
		m.File,
		[]byte(m.Content),
		0644,
	)

	if err != nil {

		return fmt.Errorf(
			"ошибка сохранения файла: %w",
			err,
		)
	}

	return nil
}
