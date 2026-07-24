package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// FindNfoFiles ищет все NFO файлы в указанной папке и её подкаталогах
func FindNfoFiles(root string) ([]string, error) {
	var files []string

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIdx := 0
	lastUpdate := time.Now()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.EqualFold(filepath.Ext(path), ".nfo") {
			files = append(files, path)
		}

		// Обновляем анимацию не чаще, чем раз в 80 мс
		if time.Since(lastUpdate) > 80*time.Millisecond {
			fmt.Printf("\r%s %s %s", frames[frameIdx], T("searching_nfo"), TF("nfo_found", len(files)))
			frameIdx = (frameIdx + 1) % len(frames)
			lastUpdate = time.Now()
		}

		return nil
	})

	if err != nil {
		fmt.Print("\r\033[K")
		return nil, fmt.Errorf("%s: %w", T("err_search_nfo"), err)
	}

	fmt.Print("\r\033[K")

	return files, nil
}
