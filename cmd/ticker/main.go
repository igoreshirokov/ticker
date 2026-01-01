package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"

	"website-checker/internal/app"
	"website-checker/internal/checker"
	"website-checker/internal/config"
)

// Глобальные переменные для управления
var (
	cfg          *config.Config
	checking        bool
	lastCheckTime   time.Time
	lastCheckResult string
	stopChan        chan bool
	mutex           sync.RWMutex
	verbose         bool
	configFile 		*string
	appName = 		"Мониторинг сайтов"
)

var iconBad = app.IconBad
var iconGood = app.IconGood



func main() {
	currentDir, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	currentDir = filepath.Dir(currentDir)
	defaultConfig := filepath.Join(currentDir, "config.yml")
	configFile = flag.String("config", defaultConfig, "Путь к файлу конфигурации")
	verboseFlag := flag.Bool("v", false, "Подробный вывод")

	flag.Parse()
	beeep.AppName = appName

	// Загрузка конфигурации
	cfg, err = config.Load(*configFile)
	if err != nil {
		beeep.Alert("Проверка конфигурации", "Ошибка загрузки: \n" + err.Error(), "")
		os.Exit(1)
	}

	beeep.Notify("Проверка конфигурации", fmt.Sprintf("Загружено %d сайтов для проверки\n", len(cfg.Sites)), "")
	// Сохраняем флаг verbose
	verbose = *verboseFlag

	// Запуск в режиме с иконкой в трее
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
	systray.SetIcon(iconGood)
	systray.SetTitle("Проверка доступности")
	systray.SetTooltip("Мониторинг сайтов")

	// Добавляем пункты меню
	mCheckNow := systray.AddMenuItem("Проверить сейчас", "Выполнить проверку")
	mStatus := systray.AddMenuItem("Статус: Не проверялось", "Последний статус")
	mStatus.Disable()
	
	systray.AddSeparator()
	
	mSettings := systray.AddMenuItem("Настройки", "Открыть конфигурацию")
	mViewLog := systray.AddMenuItem("Просмотр лога", "Показать историю проверок")
	
	systray.AddSeparator()
	
	mPause := systray.AddMenuItem("Пауза", "Приостановить проверки")
	mRestart := systray.AddMenuItem("Перезапустить", "Перезапустить мониторинг")
	mQuit := systray.AddMenuItem("Выход", "Завершить программу")

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
				
				results := checker.CheckAllSites(cfg, verbose)
				updateStatus(results, mStatus)
				
				mutex.Lock()
				checking = false
				mutex.Unlock()
				
			case <-mSettings.ClickedCh:
				openConfigFile()
				
			case <-mViewLog.ClickedCh:
				showLog()
				
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
	results := checker.CheckAllSites(cfg, verbose)
	updateStatus(results, statusItem)
	
	for {
		select {
		case <-ticker.C:
			mutex.RLock()
			isChecking := checking
			mutex.RUnlock()
			
			if !isChecking {
				results := checker.CheckAllSites(cfg, verbose)
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
		systray.SetIcon(iconGood)
		statusItem.SetIcon(iconGood)
		statusItem.SetTitle(fmt.Sprintf("✅ OK (%s)", lastCheckTime.Format("15:04")))
		if cfg.Notifications.ShowPopup {
			sendSuccessNotification(cfg)
		}
	} else {
		systray.SetIcon(iconBad)
		statusItem.SetIcon(iconBad)
		statusItem.SetTitle(fmt.Sprintf("⚠️ %d ошибок (%s)", len(failed), lastCheckTime.Format("15:04")))
		if cfg.Notifications.ShowPopup {
			sendFailNotification(cfg, failed)
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

func showLog() {
	// Показать лог проверок
	beeep.Alert("История проверок", lastCheckResult, "")
}

func togglePause(menuItem *systray.MenuItem) {
	mutex.Lock()
	checking = !checking
	if checking {
		menuItem.SetTitle("Возобновить")
	} else {
		menuItem.SetTitle("Пауза")
	}
	mutex.Unlock()
}

func restartApp() {
	// Перезапуск приложения
	exe, _ := os.Executable()
	exec.Command(exe).Start()
	os.Exit(0)
}

func sendSuccessNotification(config *config.Config) {
	if !config.Notifications.ShowPopup {
		return
	}
	
	beeep.Notify("Проверка доступности", "✅ Все сайты работают нормально!", iconGood)
}

func sendFailNotification(config *config.Config, failedResults []checker.CheckResult) {
	if !config.Notifications.ShowPopup || len(failedResults) == 0 {
		return
	}
	beeep.AppName = appName
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
	beeep.Alert(title, msg, iconBad)
}
