package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	versionFlag := flag.Bool("v", false, "Показать версию программы")
	helpFlag := flag.Bool("h", false, "Показать эту справку")

	flag.Usage = func() {

		name := filepath.Base(os.Args[0])

		fmt.Println()

		fmt.Println("Emby NFO Fixer")
		fmt.Printf(
			"Версия %s • %s\n",
			Version,
			VersionDate,
		)

		fmt.Println()

		fmt.Println("Использование:")
		fmt.Printf(
			"  %-12s %s %s %s\n",
			name,
			"[OPTIONS]",
			"[SOURCE]",
			"[BACKUP]",
		)

		fmt.Println()

		fmt.Println("Аргументы:")

		fmt.Printf(
			"  %-12s Путь к папке с фильмами\n",
			"SOURCE",
		)

		fmt.Printf(
			"  %-12s Путь к папке для резервного копирования\n",
			"BACKUP",
		)

		fmt.Println()

		fmt.Println("Опции:")

		fmt.Printf(
			"  %-12s Показать эту справку\n",
			"-h",
		)

		fmt.Printf(
			"  %-12s Показать версию программы\n",
			"-v",
		)

		fmt.Println()

		fmt.Println("Пример:")
		fmt.Printf(
			"  %-12s /mnt/Movies /mnt/Backups\n",
			name,
		)

		fmt.Println()
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *versionFlag {

		fmt.Println("Emby NFO Fixer")

		fmt.Printf(
			"Версия %s • %s\n",
			Version,
			VersionDate,
		)

		return
	}

	// Инициализируем TMDB (создаст emby_nfo_fixer.conf, если его еще нет)
	tmdbClient, tmdbErr := NewTMDBClient()

	args := flag.Args()

	if len(args) != 2 {

		flag.Usage()
		os.Exit(1)
	}

	source := args[0]
	backup := args[1]

	logger, err := NewLogger(backup)

	if err != nil {

		fmt.Println(err.Error())
		os.Exit(1)
	}

	defer logger.Close()

	fmt.Println()
	fmt.Println("Emby NFO Fixer")
	fmt.Printf(
		"Версия %s • %s\n",
		Version,
		VersionDate,
	)
	fmt.Println()

	// Проверка доступности TMDB API
	tmdbAvailable := true

	if tmdbErr != nil {
		tmdbAvailable = false
		logger.Error("TMDB_API", fmt.Errorf("TMDB API отключено: %w", tmdbErr))
	} else if err := tmdbClient.CheckAvailability(); err != nil {
		tmdbAvailable = false
		logger.Error("TMDB_API", fmt.Errorf("TMDB API недоступно: %w", err))
	}

	fmt.Println("Источник:")
	fmt.Println(source)

	fmt.Println()

	fmt.Println("Резервная копия:")
	fmt.Println(backup)

	fmt.Println()

	files, err := FindNfoFiles(source)

	if err != nil {

		logger.Error(source, err)

		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Printf("Найдено NFO файлов: %d\n\n", len(files))

	fmt.Println("Проверка и обработка файлов...")
	fmt.Println()

	progress := NewProgress(len(files))

	needFix := 0
	skipped := 0
	createdBackups := 0
	premieredAdded := 0

	for _, file := range files {

		movie, err := LoadMovie(file)

		if err != nil {

			logger.Error(
				file,
				err,
			)

			progress.Error()
			progress.Render()

			continue
		}

		// Проверка необходимости вставки <premiered>
		if !movie.HasPremiered && tmdbAvailable {
			if movie.TmdbID != "" {
				premiereDate, err := tmdbClient.GetReleaseDate(movie.TmdbID)
				if err != nil {
					logger.Error(file, fmt.Errorf("ошибка получения даты для TMDB ID %s: %w", movie.TmdbID, err))
				} else if premiereDate != "" {
					movie.NeedsPremieredFix = true
					movie.PremieredDate = premiereDate
					movie.NeedsFix = true
				}
			} else {
				logger.Error(file, fmt.Errorf("отсутствует <premiered> и не найден <tmdbid>"))
			}
		}

		if !movie.NeedsFix {

			skipped++

			logger.Skip(
				file,
				movie,
			)

			progress.Success()
			progress.Render()

			continue
		}

		needFix++

		backupFile, err := CreateBackup(
			file,
			source,
			backup,
		)

		if err != nil {

			logger.Error(
				file,
				err,
			)

			progress.Error()
			progress.Render()

			continue
		}

		createdBackups++

		movie.Fix()

		err = movie.Save()

		if err != nil {

			logger.Error(
				file,
				err,
			)

			progress.Error()
			progress.Render()

			continue
		}

		if movie.PremieredDate != "" {
			premieredAdded++
		}

		logger.Changed(
			file,
			backupFile,
			movie,
		)

		progress.Success()
		progress.Render()
	}

	progress.Finish()

	fmt.Println()

	// Ширина таблицы 40 символов
	width := 40
	line := strings.Repeat("─", width-2)

	fmt.Printf("┌%s┐\n", line)
	fmt.Printf("│ %-36s │\n", "ИТОГИ ОБРАБОТКИ")
	fmt.Printf("├%s┤\n", line)
	fmt.Printf("│ Всего файлов:     %-18d │\n", len(files))
	fmt.Printf("│ Исправлено:       %-18d │\n", needFix)
	fmt.Printf("│ Добавлена дата:   %-18d │\n", premieredAdded)
	fmt.Printf("│ Пропущено:        %-18d │\n", skipped)
	fmt.Printf("│ Резервных копий:  %-18d │\n", createdBackups)

	if progress.errors > 0 {
		fmt.Printf("│ Ошибок:           %-18d │\n", progress.errors)
	}

	fmt.Printf("└%s┘\n", line)

	fmt.Println()

	if logger.HasChanged() || logger.HasSkipped() || logger.HasErrors() {
		fmt.Println("Логи операций:")

		var activeLogs []string
		if logger.HasChanged() {
			activeLogs = append(activeLogs, filepath.Join(backup, "changed.log"))
		}
		if logger.HasSkipped() {
			activeLogs = append(activeLogs, filepath.Join(backup, "skipped.log"))
		}
		if logger.HasErrors() {
			activeLogs = append(activeLogs, filepath.Join(backup, "error.log"))
		}

		for i, logPath := range activeLogs {
			if i == len(activeLogs)-1 {
				fmt.Printf(" └─ %s\n", logPath)
			} else {
				fmt.Printf(" ├─ %s\n", logPath)
			}
		}
	} else {
		fmt.Println("Логи не создавались (нет записей).")
	}

	fmt.Println()
}
