package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var Version = "dev"
var VersionDate = "unknown"

func main() {

	// 1. Загружаем конфиг и устанавливаем язык
	cfg, cfgErr := EnsureConfig()
	SetLanguage(cfg.Language)

	// 2. Объявляем флаги
	versionFlag := flag.Bool("v", false, T("flag_version_desc"))
	helpFlag := flag.Bool("h", false, T("flag_help_desc"))
	embyFlag := flag.Bool("e", false, T("flag_emby_desc"))
	silentFlag := flag.Bool("s", false, T("flag_silent_desc"))

	flag.Usage = func() {
		name := filepath.Base(os.Args[0])

		fmt.Println()

		fmt.Println(T("usage_title"))
		fmt.Println(TF("usage_ver", Version, VersionDate))
		fmt.Println(TF("usage_config", GetConfigPath()))

		fmt.Println()

		fmt.Println(T("usage_usage"))
		fmt.Printf(
			"  %-12s %s %s %s\n",
			name,
			"[OPTIONS]",
			"<SOURCE>",
			"<BACKUP>",
		)

		fmt.Println()

		fmt.Println(T("usage_args"))

		fmt.Printf(
			"  %-12s %s\n",
			"SOURCE",
			T("usage_arg_source"),
		)

		fmt.Printf(
			"  %-12s %s\n",
			"BACKUP",
			T("usage_arg_backup"),
		)

		fmt.Println()

		fmt.Println(T("usage_options"))

		fmt.Printf(
			"  %-12s %s\n",
			"-e",
			T("flag_emby_desc"),
		)

		fmt.Printf(
			"  %-12s %s\n",
			"-s",
			T("flag_silent_desc"),
		)

		fmt.Printf(
			"  %-12s %s\n",
			"-v",
			T("flag_version_desc"),
		)

		fmt.Printf(
			"  %-12s %s\n",
			"-h",
			T("flag_help_desc"),
		)

		fmt.Println()

		fmt.Println(T("usage_example"))
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
			fmt.Println()
			fmt.Println(T("usage_title"))
			fmt.Println(TF("usage_ver", Version, VersionDate))
			fmt.Println(TF("usage_config", GetConfigPath()))
			fmt.Println()
		}
		return
	}

	// 3. Выводим ошибку загрузки конфига (если была), когда убедились, что это не был вызов -h или -v
	if cfgErr != nil && !*silentFlag {
		fmt.Println(TF("err_config", cfgErr))
	}

	// 4. Инициализируем TMDB клиент
	tmdbClient := NewTMDBClient(cfg.TmdbToken)

	// 5. Если указан флаг -e, проверяем наличие настроек Emby
	if *embyFlag {
		if cfg.EmbyURL == "" || cfg.EmbyApiKey == "" {
			if !*silentFlag {
				fmt.Println(T("warn_emby_no_auth"))
			}
			serverURL, apiKey := PromptForEmbyInteractive()
			if serverURL == "" || apiKey == "" {
				if !*silentFlag {
					fmt.Println(T("err_emby_no_auth"))
				}
				os.Exit(1)
			}
			cfg.EmbyURL = serverURL
			cfg.EmbyApiKey = apiKey
			_ = SaveConfig(cfg)
		}
	}

	// 6. Проверяем аргументы запуска (SOURCE и BACKUP)
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
		fmt.Println(T("usage_title"))
		fmt.Println(TF("usage_ver", Version, VersionDate))
		fmt.Println()
	}

	// Проверка доступности TMDB API
	tmdbAvailable := true

	if err := tmdbClient.CheckAvailability(); err != nil {
		tmdbAvailable = false
		logger.Error("TMDB_API", fmt.Errorf(T("err_tmdb_unavail"), err))
	}

	if !*silentFlag {
		fmt.Println(T("source_dir"))
		fmt.Println(source)

		fmt.Println()

		fmt.Println(T("backup_dir"))
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
		fmt.Println(TF("nfo_found", len(files)))
		fmt.Println(T("processing"))
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
		fmt.Printf("│ %-36s │\n", T("summary_title"))
		fmt.Printf("├%s┤\n", line)
		fmt.Printf("│ %-18s %-17d │\n", T("total_files"), len(files))
		fmt.Printf("│ %-18s %-17d │\n", T("fixed_files"), needFix)
		fmt.Printf("│ %-18s %-17d │\n", T("premiered_added"), premieredAdded)
		fmt.Printf("│ %-18s %-17d │\n", T("skipped_files"), skipped)
		fmt.Printf("│ %-18s %-17d │\n", T("backups_created"), createdBackups)

		if progress.errors > 0 {
			fmt.Printf("│ %-18s %-17d │\n", T("error_count"), progress.errors)
		}

		fmt.Printf("└%s┘\n", line)

		fmt.Println()

		if logger.HasChanged() || logger.HasSkipped() || logger.HasErrors() {
			fmt.Println(T("log_header_title"))

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
			fmt.Println(T("no_logs_created"))
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
					fmt.Println(T("emby_scan_error"))
				}
			} else {
				if !*silentFlag {
					fmt.Println(T("emby_scan_started"))
					fmt.Println(T("emby_scan_wait"))
					fmt.Println()
				}
			}
		} else {
			if !*silentFlag {
				fmt.Println(T("emby_scan_skipped"))
				fmt.Println()
			}
		}
	}
}
