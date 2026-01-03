package systray

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
	"website-checker/internal/app"
	"website-checker/internal/checker"
	"website-checker/internal/config"
	"website-checker/internal/i18n"
	"website-checker/internal/notification"

	"github.com/getlantern/systray"
)

var (
	stopChan        chan bool
	mutex           sync.RWMutex
	checking        bool
	cfg				*config.Config
	lastCheckTime   time.Time
	lastCheckResult string
	configFile 		*string
)

func Run(globalConfig *config.Config, configFilePath string) {
	cfg = globalConfig
	configFile = &configFilePath
	runWithTray()
}

func runWithTray() {
	// Канал для остановки
	stopChan = make(chan bool)

	// Запускаем системный трей
	systray.Run(onReady, onExit)
}

func onReady() {
	// Устанавливаем иконку
	systray.SetIcon(app.IconGood)
	// Переводим заголовок
	systray.SetTitle(i18n.T("checker_title"))
	// Переводим подсказку
	systray.SetTooltip(i18n.T("checker_tooltip"))

	// Переводим пункты меню
	mCheckNow := systray.AddMenuItem(i18n.T("checker_check_now"), i18n.T("checker_check_now_tooltip"))
	mStatus := systray.AddMenuItem(i18n.T("checker_status_not_checked"), i18n.T("checker_status_not_checked_tooltip"))
	mStatus.Disable()

	systray.AddSeparator()

	// Переводим пункты меню
	mSettings := systray.AddMenuItem(i18n.T("checker_settings"), i18n.T("checker_settings_tooltip"))
	mViewLog := systray.AddMenuItem(i18n.T("checker_view_log"), i18n.T("checker_view_log_tooltip"))

	systray.AddSeparator()

	// Переводим пункты меню
	mPause := systray.AddMenuItem(i18n.T("checker_pause"), i18n.T("checker_pause_tooltip"))
	mRestart := systray.AddMenuItem(i18n.T("checker_restart"), i18n.T("checker_restart_tooltip"))
	mQuit := systray.AddMenuItem(i18n.T("checker_quit"), i18n.T("checker_quit_tooltip"))

	// Запускаем фоновую проверку
	go backgroundChecker(mStatus)

	// Обработка событий меню
	go func() {
		for {
			select {
			case <-mCheckNow.ClickedCh:
				mutex.Lock()
				checking = true
				mutex.Unlock()

				results := checker.CheckAllSites(cfg)
				updateStatus(results, mStatus)

				mutex.Lock()
				checking = false
				mutex.Unlock()

			case <-mSettings.ClickedCh:
				openConfigFile()

			case <-mViewLog.ClickedCh:
				notification.ShowLog(lastCheckResult)

			case <-mPause.ClickedCh:
				togglePause(mPause)

			case <-mRestart.ClickedCh:
				restartApp()

			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Очистка ресурсов
	if stopChan != nil {
		close(stopChan)
	}
}

func backgroundChecker(statusItem *systray.MenuItem) {
	ticker := time.NewTicker(time.Duration(cfg.General.CheckInterval) * time.Second)
	defer ticker.Stop()

	// Первая проверка сразу при старте
	results := checker.CheckAllSites(cfg)
	updateStatus(results, statusItem)

	for {
		select {
		case <-ticker.C:
			mutex.RLock()
			isChecking := checking
			mutex.RUnlock()

			if !isChecking {
				results := checker.CheckAllSites(cfg)
				updateStatus(results, statusItem)
			}

		case <-stopChan:
			return
		}
	}
}

func updateStatus(results []checker.CheckResult, statusItem *systray.MenuItem) {
	lastCheckTime = time.Now()

	failed := getFailedResults(results)
	allOK := len(failed) == 0

	// Обновляем иконку в зависимости от статуса
	if allOK {
		systray.SetIcon(app.IconGood)
		statusItem.SetIcon(app.IconGood)
		// Переводим статус
		statusItem.SetTitle(fmt.Sprintf(i18n.T("checker_status_ok"), lastCheckTime.Format("15:04")))
		if cfg.Notifications.ShowPopup {
			notification.SendSuccess()
		}
	} else {
		systray.SetIcon(app.IconBad)
		statusItem.SetIcon(app.IconBad)
		// Переводим статус
		statusItem.SetTitle(i18n.T("checker_status_error", len(failed), lastCheckTime.Format("15:04")))
		if cfg.Notifications.ShowPopup {
			notification.SendFail(failed)
		}
	}

	// Сохраняем результат для просмотра
	mutex.Lock()
	lastCheckResult = formatResults(results)
	mutex.Unlock()
}

func getFailedResults(results []checker.CheckResult) []checker.CheckResult {
	var failed []checker.CheckResult
	for _, result := range results {
		if !result.Success {
			failed = append(failed, result)
		}
	}
	return failed
}

func formatResults(results []checker.CheckResult) string {
	var output string
	for _, result := range results {
		status := "✅"
		if !result.Success {
			status = "❌"
		}
		output += fmt.Sprintf("%s %s: %d (%v)\n",
			status, result.Site.Name, result.StatusCode, result.Duration)
	}
	return output
}

// Вспомогательные функции
func openConfigFile() {
	// Открыть файл конфигурации в блокноте
	exec.Command("notepad.exe", *configFile).Start()
}

func togglePause(menuItem *systray.MenuItem) {
	mutex.Lock()
	checking = !checking
	if checking {
		// Переводим пункт меню
		menuItem.SetTitle(i18n.T("checker_resume"))
	} else {
		// Переводим пункт меню
		menuItem.SetTitle(i18n.T("checker_pause"))
	}
	mutex.Unlock()
}

func restartApp() {
	// Перезапуск приложения
	exe, _ := os.Executable()
	exec.Command(exe).Start()
	os.Exit(0)
}
