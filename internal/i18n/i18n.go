package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales/*.json
var localesFS embed.FS

var translations map[string]string

// Load загружает переводы для указанного языка.
// Если язык не найден или не указан, по умолчанию используется английский.
func Load(lang string) error {
	if lang == "" {
		lang = "en" // Язык по умолчанию
	}

	fileName := fmt.Sprintf("locales/%s.json", lang)
	data, err := localesFS.ReadFile(fileName)
	if err != nil {
		// Если файл для языка не найден, пробуем загрузить английский
		data, err = localesFS.ReadFile("locales/en.json")
		if err != nil {
			return fmt.Errorf("failed to load default translation file: %w", err)
		}
	}

	err = json.Unmarshal(data, &translations)
	if err != nil {
		return fmt.Errorf("failed to parse translation file %s: %w", fileName, err)
	}

	return nil
}

// T возвращает переведенную строку по ключу.
// Дополнительные аргументы используются для форматирования строки.
// Пример: T("statusError", "count", 5) вернет "⚠️ 5 ошибок (...)"
func T(key string, args ...interface{}) string {
	translation, ok := translations[key]
	if !ok {
		// Если перевод не найден, возвращаем ключ
		return key
	}

	// Заменяем плейсхолдеры вида {key} на значения
	if len(args) > 0 && len(args)%2 == 0 {
		// Создаем слайс для пар "старая строка, новая строка"
		var replacerArgs []string
		for i := 0; i < len(args); i += 2 {
			// Формируем плейсхолдер, например "{count}"
			placeholder := fmt.Sprintf("{%v}", args[i])
			// Формируем значение, например "5"
			value := fmt.Sprintf("%v", args[i+1])
			// Добавляем пару в слайс
			replacerArgs = append(replacerArgs, placeholder, value)
		}
		// Создаем Replacer со всеми парами и выполняем замену
		return strings.NewReplacer(replacerArgs...).Replace(translation)
	}

	return translation
}