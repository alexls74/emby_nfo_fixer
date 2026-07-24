package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Logger struct {
	changed *os.File
	skip    *os.File
	errors  *os.File

	backupDir string

	hasChanged bool
	hasSkipped bool
	hasErrors  bool
}

func NewLogger(backupDir string) (*Logger, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf(
			"%s: %w",
			T("err_create_log_dir"),
			err,
		)
	}

	logger := &Logger{
		backupDir: backupDir,
	}

	return logger, nil
}

func (l *Logger) writeHeader(file *os.File) {
	now := time.Now().Format("2006-01-02 15:04:05")

	header := fmt.Sprintf(
		"========== %s %s ==========\n\n",
		T("log_header_title"),
		now,
	)

	fmt.Fprint(file, header)
}

func (l *Logger) Skip(
	file string,
	movie *Movie,
) {
	if l.skip == nil {
		skipFile, err := os.OpenFile(
			filepath.Join(l.backupDir, "skipped.log"),
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			0644,
		)
		if err != nil {
			return
		}
		l.skip = skipFile
		l.writeHeader(l.skip)
	}

	l.hasSkipped = true

	// Явно передаем "%s" как константный формат
	fmt.Fprintf(
		l.skip,
		"%s",
		TF("log_skip_reason", file),
	)
}

func (l *Logger) Changed(
	source string,
	backup string,
	movie *Movie,
) {
	if l.changed == nil {
		changedFile, err := os.OpenFile(
			filepath.Join(l.backupDir, "changed.log"),
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			0644,
		)
		if err != nil {
			return
		}
		l.changed = changedFile
		l.writeHeader(l.changed)
	}

	l.hasChanged = true

	var details []string
	if movie.MovedCredits > 0 || movie.MovedDirectors > 0 {
		details = append(details, TF("log_moved_details", movie.MovedCredits, movie.MovedDirectors))
	}
	if movie.PremieredDate != "" {
		details = append(details, TF("log_added_date", movie.PremieredDate))
	}

	// Явно передаем "%s" как константный формат
	fmt.Fprintf(
		l.changed,
		"%s",
		TF("log_changed_title", source, backup, strings.Join(details, "\n")),
	)
}

func (l *Logger) Error(
	file string,
	err error,
) {
	if l.errors == nil {
		errorFile, openErr := os.OpenFile(
			filepath.Join(
				l.backupDir,
				"error.log",
			),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)

		if openErr != nil {
			return
		}

		l.errors = errorFile
	}

	l.hasErrors = true

	now := time.Now().Format("2006-01-02 15:04:05")

	fmt.Fprintf(
		l.errors,
		"%s\nFILE:\n%s\nERROR:\n%s\n\n",
		now,
		file,
		err,
	)
}

func (l *Logger) HasChanged() bool {
	return l.hasChanged
}

func (l *Logger) HasSkipped() bool {
	return l.hasSkipped
}

func (l *Logger) HasErrors() bool {
	return l.hasErrors
}

func (l *Logger) Close() {
	if l.changed != nil {
		l.changed.Close()
	}

	if l.skip != nil {
		l.skip.Close()
	}

	if l.errors != nil {
		l.errors.Close()
	}
}
