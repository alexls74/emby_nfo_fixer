package main

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed locales/*.json
var localesFS embed.FS

type Key string

var (
	translations = make(map[string]map[Key]string)
	currentLang  = "ru"
)

func init() {
	loadLocale("ru")
	loadLocale("en")
}

func loadLocale(lang string) {
	data, err := localesFS.ReadFile(fmt.Sprintf("locales/%s.json", lang))
	if err != nil {
		return
	}
	var dict map[Key]string
	if err := json.Unmarshal(data, &dict); err == nil {
		translations[lang] = dict
	}
}

// SetLanguage устанавливает текущий язык локализации
func SetLanguage(lang string) {
	if lang == "en" || lang == "ru" {
		currentLang = lang
	}
}

// GetLanguage возвращает текущий язык
func GetLanguage() string {
	return currentLang
}

// T возвращает переведенную строку по ключу
func T(key Key) string {
	if dict, exists := translations[currentLang]; exists {
		if msg, ok := dict[key]; ok {
			return msg
		}
	}
	// Fallback на русский
	if dict, exists := translations["ru"]; exists {
		if msg, ok := dict[key]; ok {
			return msg
		}
	}
	return string(key)
}

// TF возвращает форматированную переведенную строку
func TF(key Key, a ...interface{}) string {
	return fmt.Sprintf(T(key), a...)
}
