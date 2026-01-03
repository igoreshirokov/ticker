package i18n

import (
	"testing"
)

func TestTranslation(t *testing.T) {
	// Шаг 1: Загружаем переводы для теста (например, русский)
	err := Load("ru")
	if err != nil {
		// t.Fatalf прерывает тест и выводит ошибку
		t.Fatalf("Не удалось загрузить файл перевода ru.json: %v", err)
	}

	// Шаг 2: Определяем тестовые случаи
	testCases := []struct {
		name     string // Название теста
		key      string // Ключ для перевода
		args     []interface{} // Аргументы для форматирования
		expected string // Ожидаемый результат
	}{
		{
			name:     "Простой перевод без аргументов",
			key:      "menuQuit",
			args:     nil,
			expected: "Выход",
		},
		{
			name:     "Перевод с одним аргументом",
			key:      "statusOk",
			args:     []interface{}{"time", "12:30"},
			expected: "✅ OK (12:30)",
		},
		{
			name:     "Перевод с несколькими аргументами",
			key:      "statusError",
			args:     []interface{}{"count", 5, "time", "15:00"},
			expected: "⚠️ 5 ошибок (15:00)",
		},
		{
			name:     "Несуществующий ключ",
			key:      "nonExistentKey",
			args:     nil,
			expected: "nonExistentKey", // Должен вернуться сам ключ
		},
	}

	// Шаг 3: Прогоняем тесты
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := T(tc.key, tc.args...)
			if got != tc.expected {
				// t.Errorf выводит ошибку, но не прерывает выполнение других тестов
				t.Errorf("Ожидали получить '%s', но получили '%s'", tc.expected, got)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Тест на загрузку языка по умолчанию (английского)
	t.Run("Загрузка языка по умолчанию", func(t *testing.T) {
		err := Load("non_existent_lang") // Указываем несуществующий язык
		if err != nil {
			t.Fatalf("Не удалось загрузить язык по умолчанию: %v", err)
		}
		
		expected := "Quit"
		got := T("menuQuit")
		if got != expected {
			t.Errorf("Ожидали '%s' из файла по умолчанию, но получили '%s'", expected, got)
		}
	})
}