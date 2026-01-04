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

func T(key string, args ...interface{}) string {
	translation, ok := translations[key]
	if !ok {
		return key
	}

	if len(args) > 0 && len(args)%2 == 0 {
		var replacerArgs []string
		for i := 0; i < len(args); i += 2 {
			placeholder := fmt.Sprintf("{%v}", args[i])
			value := fmt.Sprintf("%v", args[i+1])
			replacerArgs = append(replacerArgs, placeholder, value)
		}
		return strings.NewReplacer(replacerArgs...).Replace(translation)
	}

	return translation
}