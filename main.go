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
	embyFlag := flag.Bool("e", false, "Запустить сканирование медиатеки Emby после обработки")
	silentFlag := flag.Bool("s", false, "Тихий режим (без вывода сообщений в консоль)")

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
			"  %-12s Запустить сканирование медиатеки Emby при наличии изменений\n",
			"-e",
		)

		fmt.Printf(
			"  %-12s Тихий режим (без вывода в консоль)\n",
			"-s",
		)

		fmt.Printf(
			"  %-12s Показать версию программы\n",
			"-v",
		)

		fmt.Printf(
			"  %-12s Показать эту справку\n",
			"-h",
		)

		fmt.Println()

		fmt.Println("Пример:")
		fmt.Printf(
			"  %-12s [-e] [-s] /mnt/Movies /mnt/Backups\n",
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
		if !*silentFlag {
			fmt.Println("Emby NFO Fixer")
			fmt.Printf(
				"Версия %s • %s\n",
				Version,
				VersionDate,
			)
		}
		return
	}

	// Инициализируем TMDB (создаст emby_nfo_fixer.conf, если его еще нет)
	tmdbClient, tmdbErr := NewTMDBClient()

	// Загружаем конфиг для работы с Emby
	cfg, _ := LoadConfig()

	// Если указан флаг -e, проверяем наличие настроек Emby
	if *embyFlag {
		if cfg.EmbyURL == "" || cfg.EmbyApiKey == "" {
			if !*silentFlag {
				fmt.Println("⚠️  Указан флаг -e, но данные Emby отсутствуют в конфигурации.")
			}
			serverURL, apiKey := PromptForEmbyInteractive()
			if serverURL == "" || apiKey == "" {
				if !*silentFlag {
					fmt.Println("❌ Ошибка: Запуск сканирования Emby невозможно выполнить без настроек.")
				}
				os.Exit(1)
			}
			cfg.EmbyURL = serverURL
			cfg.EmbyApiKey = apiKey
			_ = SaveConfig(cfg)
		}
	}

	args := flag.Args()

	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	source := args[0]
	backup := args[1]

	logger, err := NewLogger(backup)

	if err != nil {
		if !*silentFlag {
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	defer logger.Close()

	if !*silentFlag {
		fmt.Println()
		fmt.Println("Emby NFO Fixer")
		fmt.Printf(
			"Версия %s • %s\n",
			Version,
			VersionDate,
		)
		fmt.Println()
	}

	// Проверка доступности TMDB API
	tmdbAvailable := true

	if tmdbErr != nil {
		tmdbAvailable = false
		logger.Error("TMDB_API", fmt.Errorf("TMDB API отключено: %w", tmdbErr))
	} else if err := tmdbClient.CheckAvailability(); err != nil {
		tmdbAvailable = false
		logger.Error("TMDB_API", fmt.Errorf("TMDB API недоступно: %w", err))
	}

	if !*silentFlag {
		fmt.Println("Источник:")
		fmt.Println(source)

		fmt.Println()

		fmt.Println("Резервная копия:")
		fmt.Println(backup)

		fmt.Println()
	}

	files, err := FindNfoFiles(source)

	if err != nil {
		logger.Error(source, err)
		if !*silentFlag {
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	if !*silentFlag {
		fmt.Printf("Найдено NFO файлов: %d\n\n", len(files))
		fmt.Println("Проверка и обработка файлов...")
		fmt.Println()
	}

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
			if !*silentFlag {
				progress.Render()
			}

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
			}
		}

		if !movie.NeedsFix {

			skipped++

			logger.Skip(
				file,
				movie,
			)

			progress.Success()
			if !*silentFlag {
				progress.Render()
			}

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
			if !*silentFlag {
				progress.Render()
			}

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
			if !*silentFlag {
				progress.Render()
			}

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
		if !*silentFlag {
			progress.Render()
		}
	}

	if !*silentFlag {
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

	// Вызов сканирования Emby при наличии флага -e и изменённых файлов
	if *embyFlag {
		if logger.HasChanged() {
			embyClient := NewEmbyClient(cfg.EmbyURL, cfg.EmbyApiKey)
			if err := embyClient.TriggerLibraryScan(); err != nil {
				logger.Error("EMBY_API", fmt.Errorf("не удалось запустить сканирование медиатеки: %w", err))
				if !*silentFlag {
					fmt.Println("⚠️ Не удалось запустить сканирование Emby (подробности в error.log)")
				}
			} else {
				if !*silentFlag {
					fmt.Println("Запущено сканирование медиатеки Emby.")
					fmt.Println("Cканирование займёт некоторое время. Пожалуйста, подождите.")
					fmt.Println()
				}
			}
		} else {
			if !*silentFlag {
				fmt.Println("ℹ️ Файлы не изменялись, сканирование Emby пропущено.")
			}
		}
	}
}
