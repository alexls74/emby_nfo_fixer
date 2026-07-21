.PHONY: build clean help check-upx check-gh release publish

BINARY_FOLDER := ./build/bin
RELEASE_FOLDER := ./release
BINARY_NAME := emby_nfo_fixer
VERSION := $(shell cat VERSION)
VERSION_DATE := $(shell date -u +%Y-%m-%d)

LDFLAGS := -ldflags "-s -w -buildid= -X main.Version=$(VERSION) -X main.VersionDate=$(VERSION_DATE)"

help:
	@echo "Использование: make [цель]"
	@echo ""
	@echo "Цели:"
	@echo "  build    Скомпилировать бинарники для всех платформ"
	@echo "  release  Скомпилировать и упаковать архивы для GitHub Release"
	@echo "  publish  Опубликовать релиз на GitHub (требует gh CLI)"
	@echo "  clean    Очистить скомпилированные файлы и релизы"
	@echo "  help     Показать это сообщение помощи"

check-upx:
	@which upx > /dev/null 2>&1 || (echo "⚠️  Внимание: UPX не установлен. Сжатие будет пропущено." && exit 1)

check-gh:
	@which gh > /dev/null 2>&1 || (echo "❌ Ошибка: GitHub CLI (gh) не установлен. Установите: https://cli.github.com/" && exit 1)
	@gh auth status > /dev/null 2>&1 || (echo "❌ Ошибка: Вы не авторизованы в gh. Выполните: gh auth login" && exit 1)

build:
	@mkdir -p ${BINARY_FOLDER}/MacOS ${BINARY_FOLDER}/Linux ${BINARY_FOLDER}/Windows

	@echo "🔨 Сборка macOS (Universal Binary)..."
	GOOS=darwin GOARCH=amd64 go build -trimpath $(LDFLAGS) -o ${BINARY_FOLDER}/MacOS/${BINARY_NAME}_amd64 .
	GOOS=darwin GOARCH=arm64 go build -trimpath $(LDFLAGS) -o ${BINARY_FOLDER}/MacOS/${BINARY_NAME}_arm64 .
	lipo -create -output ${BINARY_FOLDER}/MacOS/${BINARY_NAME} ${BINARY_FOLDER}/MacOS/${BINARY_NAME}_amd64 ${BINARY_FOLDER}/MacOS/${BINARY_NAME}_arm64
	@rm ${BINARY_FOLDER}/MacOS/${BINARY_NAME}_amd64 ${BINARY_FOLDER}/MacOS/${BINARY_NAME}_arm64

	@echo "🔨 Сборка Linux amd64..."
	GOOS=linux GOARCH=amd64 go build -trimpath $(LDFLAGS) -o ${BINARY_FOLDER}/Linux/${BINARY_NAME} .
	@upx -9 ${BINARY_FOLDER}/Linux/${BINARY_NAME}

	@echo "🔨 Сборка Windows amd64..."
	GOOS=windows GOARCH=amd64 go build -trimpath $(LDFLAGS) -o ${BINARY_FOLDER}/Windows/${BINARY_NAME}.exe .
	@upx -9 ${BINARY_FOLDER}/Windows/${BINARY_NAME}.exe

	@echo "✅ Сборка завершена в ${BINARY_FOLDER}"

release: build
	@mkdir -p ${RELEASE_FOLDER}
	@echo "📦 Упаковка релизов..."

	# macOS Universal (.zip)
	@zip -j ${RELEASE_FOLDER}/${BINARY_NAME}_v${VERSION}_macOS_universal.zip ${BINARY_FOLDER}/MacOS/${BINARY_NAME} readme.html 2>/dev/null || \
		zip -j ${RELEASE_FOLDER}/${BINARY_NAME}_v${VERSION}_macOS_universal.zip ${BINARY_FOLDER}/MacOS/${BINARY_NAME}

	# Linux amd64 (.tar.gz)
	@cp readme.html ${BINARY_FOLDER}/Linux/ 2>/dev/null || true
	@COPYFILE_DISABLE=1 tar --exclude='._*' -czf ${RELEASE_FOLDER}/${BINARY_NAME}_v${VERSION}_linux_amd64.tar.gz -C ${BINARY_FOLDER}/Linux ${BINARY_NAME} readme.html
	@rm -f ${BINARY_FOLDER}/Linux/readme.html

	# Windows x64 (.zip)
	@zip -j ${RELEASE_FOLDER}/${BINARY_NAME}_v${VERSION}_windows_x64.zip ${BINARY_FOLDER}/Windows/${BINARY_NAME}.exe readme.html 2>/dev/null || \
		zip -j ${RELEASE_FOLDER}/${BINARY_NAME}_v${VERSION}_windows_x64.zip ${BINARY_FOLDER}/Windows/${BINARY_NAME}.exe

	@echo "🎉 Готово! Все архивы сформированы в папке ${RELEASE_FOLDER}:"
	@ls -lh ${RELEASE_FOLDER}

publish: check-gh release
	@echo "🚀 Публикация релиза v$(VERSION) на GitHub..."
	@if [ ! -f CHANGELOG.md ]; then echo "❌ Ошибка: CHANGELOG.md не найден!"; exit 1; fi

	# Вырезаем описание текущей версии из CHANGELOG.md во временный файл в папке релизов
	@awk '/^## \['$(VERSION)'\]/{flag=1; next} /^## \[/{flag=0} flag' CHANGELOG.md > ${RELEASE_FOLDER}/release_notes.md

	@if [ ! -s ${RELEASE_FOLDER}/release_notes.md ]; then \
		echo "⚠️ Предупреждение: В CHANGELOG.md не найдено описание для версии [$(VERSION)]. Использован стандартный текст."; \
		echo "Релиз версии v$(VERSION)" > ${RELEASE_FOLDER}/release_notes.md; \
	fi

	# Публикация релиза на GitHub
	gh release create "v$(VERSION)" ${RELEASE_FOLDER}/*.zip ${RELEASE_FOLDER}/*.tar.gz \
		--title "v$(VERSION)" \
		--notes-file ${RELEASE_FOLDER}/release_notes.md

	@rm -f ${RELEASE_FOLDER}/release_notes.md
	@echo "✨ Релиз v$(VERSION) успешно опубликован на GitHub!"

clean:
	rm -rf ${BINARY_FOLDER}
	go clean