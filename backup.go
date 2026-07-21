package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateBackup создаёт копию файла, сохраняя структуру каталогов
func CreateBackup(sourceFile, sourceRoot, backupRoot string) (string, error) {
	// Получаем относительный путь от корня SOURCE
	relativePath, err := filepath.Rel(sourceRoot, sourceFile)
	if err != nil {
		return "", fmt.Errorf("не удалось определить относительный путь: %w", err)
	}

	// Полный путь в BACKUP
	backupFile := filepath.Join(backupRoot, relativePath)

	// Создаём каталог назначения
	backupDir := filepath.Dir(backupFile)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать каталог бэкапа: %w", err)
	}

	// Копируем файл
	if err := copyFile(sourceFile, backupFile); err != nil {
		return "", fmt.Errorf("ошибка копирования %s: %w", sourceFile, err)
	}

	return backupFile, nil
}

// copyFile копирует один файл
func copyFile(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Сохраняем права файла
	info, err := os.Stat(source)
	if err == nil {
		_ = os.Chmod(destination, info.Mode())
	}

	return nil
}
