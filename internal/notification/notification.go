package notification

import (
	"fmt"
	"time"
	"website-checker/internal/app"
	"website-checker/internal/checker"
	"website-checker/internal/config"

	"github.com/gen2brain/beeep"
)

func Init(appName string) {
	beeep.AppName = appName
}

func SendSuccess(config *config.Config) {
	if !config.Notifications.ShowPopup {
		return
	}

	title := "Проверка доступности"
	msg := "✅ Все сайты работают нормально!"

	beeep.Notify(title, msg, app.IconGood)
}

func SendFail(config *config.Config, failedResults []checker.CheckResult) {
	if !config.Notifications.ShowPopup || len(failedResults) == 0 {
		return
	}
	beeep.AppName = app.AppName
	title := "Проверка доступности"
	msg := "⚠️ Обнаружены проблемы с сайтами:\n\n"
	for _, result := range failedResults {
		statusText := "ОШИБКА"
		if result.StatusCode > 0 {
			statusText = fmt.Sprintf("Статус: %d", result.StatusCode)
		}
		duration := result.Duration.Round(time.Millisecond)
		msg += fmt.Sprintf("• %s: %s (время: %v)\n",
			result.Site.Name, statusText, duration)
	}
	beeep.Alert(title, msg, app.IconBad)
}

func SendConfigLoaded(config config.Config) {
	if !config.Notifications.ShowPopup {
		return
	}
	title := "Проверка конфигурации"
	msg := fmt.Sprintf("Загружено %d сайтов для проверки\n", len(config.Sites))

	beeep.Notify(title, msg, app.IconGood)
}

func ShowLog(lastCheckResult string) {
	// Показать лог проверок
	beeep.Alert("История проверок", lastCheckResult, "")
}

func Error(title string, msg string) {
	beeep.Alert("Ошибка", msg, app.IconBad)
}