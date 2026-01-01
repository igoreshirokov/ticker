package main

import (
	"crypto/tls"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"golang.org/x/sys/windows/svc"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Sites         []SiteConfig  `yaml:"sites"`
	Notifications Notifications `yaml:"notifications"`
	General       GeneralConfig `yaml:"general"`
}

type SiteConfig struct {
	URL     string `yaml:"url"`
	Name    string `yaml:"name"`
	Timeout int    `yaml:"timeout"`
}

type Notifications struct {
	ShowPopup     bool `yaml:"show_popup"`
	ConsoleOutput bool `yaml:"console_output"`
}

type GeneralConfig struct {
	CheckInterval    int `yaml:"check_interval"`
	ConcurrentChecks int `yaml:"concurrent_checks"`
}

type CheckResult struct {
	Site       SiteConfig
	Success    bool
	StatusCode int
	Error      string
	Duration   time.Duration
}

// Глобальные переменные для управления
var (
	config          *Config
	checking        bool
	lastCheckTime   time.Time
	lastCheckResult string
	stopChan        chan bool
	mutex           sync.RWMutex
	verbose         bool
	appName = 		"Мониторинг сайтов"
)

//go:embed assets/danger.ico
var iconBad []byte
//go:embed assets/info.ico
var iconGood []byte



func main() {
	currentDir, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	currentDir = filepath.Dir(currentDir)
	defaultConfig := filepath.Join(currentDir, "config.yml")
	configFile := flag.String("config", defaultConfig, "Путь к файлу конфигурации")
	verboseFlag := flag.Bool("v", false, "Подробный вывод")

	flag.Parse()
	beeep.AppName = appName

	// Загрузка конфигурации
	config, err = loadConfig(*configFile)
	if err != nil {
		beeep.Alert("Проверка конфигурации", "Ошибка загрузки: \n" + err.Error(), "")
		os.Exit(1)
	}

	beeep.Notify("Проверка конфигурации", fmt.Sprintf("Загружено %d сайтов для проверки\n", len(config.Sites)), "")
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
				
				results := checkAllSites(config, verbose)
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
	ticker := time.NewTicker(time.Duration(config.General.CheckInterval) * time.Second)
	defer ticker.Stop()
	
	// Первая проверка сразу при старте
	results := checkAllSites(config, verbose)
	updateStatus(results, statusItem)
	
	for {
		select {
		case <-ticker.C:
			mutex.RLock()
			isChecking := checking
			mutex.RUnlock()
			
			if !isChecking {
				results := checkAllSites(config, verbose)
				updateStatus(results, statusItem)
			}
			
		case <-stopChan:
			return
		}
	}
}

func updateStatus(results []CheckResult, statusItem *systray.MenuItem) {
	lastCheckTime = time.Now()
	
	failed := getFailedResults(results)
	allOK := len(failed) == 0
	
	// Обновляем иконку в зависимости от статуса
	if allOK {
		systray.SetIcon(iconGood)
		statusItem.SetIcon(iconGood)
		statusItem.SetTitle(fmt.Sprintf("✅ OK (%s)", lastCheckTime.Format("15:04")))
		if config.Notifications.ShowPopup {
			beeep.Notify("✅ OK", fmt.Sprintf("Все сайты доступны (%s)", lastCheckTime.Format("15:04")), "")
		}
	} else {
		systray.SetIcon(iconBad)
		statusItem.SetIcon(iconBad)
		statusItem.SetTitle(fmt.Sprintf("⚠️ %d ошибок (%s)", len(failed), lastCheckTime.Format("15:04")))
		if config.Notifications.ShowPopup {
			sendFailNotification(config, failed)
		}
	}
	
	// Сохраняем результат для просмотра
	mutex.Lock()
	lastCheckResult = formatResults(results)
	mutex.Unlock()
}

func getFailedResults(results []CheckResult) []CheckResult {
	var failed []CheckResult
	for _, result := range results {
		if !result.Success {
			failed = append(failed, result)
		}
	}
	return failed
}

func formatResults(results []CheckResult) string {
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

// Функции для работы с иконками
func getIconData(path string) []byte {
	// Чтение иконки из файла
	iconData, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Ошибка чтения иконки %s: %v\n", path, err)
		return nil
	}
	return iconData
}

// Вспомогательные функции
func openConfigFile() {
	// Открыть файл конфигурации в блокноте
	exec.Command("notepad.exe", "config.yml").Start()
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

type service struct{}

func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	
	// Загрузка конфигурации
	var err error
	config, err = loadConfig("config.yml")
	if err != nil {
		return false, 1
	}
	
	// Запуск фоновой проверки
	stopChan = make(chan bool)
	go func() {
		ticker := time.NewTicker(time.Duration(config.General.CheckInterval) * time.Second)
		for {
			select {
			case <-ticker.C:
				checkAllSites(config, false)
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()
	
	changes <- svc.Status{State: svc.Running}
	
	// Ожидание команд
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				if stopChan != nil {
					close(stopChan)
				}
				return false, 0
			case svc.Interrogate:
				changes <- c.CurrentStatus
			}
		}
	}
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("файл конфигурации %s не найден", filename)
		}
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func checkAllSites(config *Config, verbose bool) []CheckResult {
	var wg sync.WaitGroup
	results := make([]CheckResult, len(config.Sites))
	semaphore := make(chan struct{}, config.General.ConcurrentChecks)

	for i, site := range config.Sites {
		wg.Add(1)
		go func(idx int, site SiteConfig) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results[idx] = checkSite(site, verbose)
		}(i, site)
	}

	wg.Wait()
	return results
}

func checkSite(site SiteConfig, verbose bool) CheckResult {
	start := time.Now()
	client := &http.Client{
		Timeout: time.Duration(site.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	req, err := http.NewRequest("GET", site.URL, nil)
	if err != nil {
		return CheckResult{
			Site:     site,
			Success:  false,
			Error:    fmt.Sprintf("Ошибка создания запроса: %v", err),
			Duration: time.Since(start),
		}
	}

	req.Header.Set("User-Agent", "WebsiteChecker/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			Site:     site,
			Success:  false,
			Error:    fmt.Sprintf("Ошибка соединения: %v", err),
			Duration: time.Since(start),
		}
	}
	defer resp.Body.Close()

	_, err = io.CopyN(io.Discard, resp.Body, 4096)
	if err != nil && err != io.EOF {
		return CheckResult{
			Site:       site,
			Success:    false,
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("Ошибка чтения ответа: %v", err),
			Duration:   time.Since(start),
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	
	if verbose {
		fmt.Printf("[DEBUG] %s: %d %s (%v)\n", 
			site.Name, resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start))
	}

	return CheckResult{
		Site:       site,
		Success:    success,
		StatusCode: resp.StatusCode,
		Error:      "",
		Duration:   time.Since(start),
	}
}

func allSitesOK(results []CheckResult) bool {
	for _, result := range results {
		if !result.Success {
			return false
		}
	}
	return true
}

func sendSuccessNotification(config *Config) {
	if !config.Notifications.ShowPopup {
		return
	}
	
	beeep.Notify("Проверка доступности", "✅ Все сайты работают нормально!", "assets/info.ico")
}

func sendFailNotification(config *Config, failedResults []CheckResult) {
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
	beeep.Alert(title, msg, "assets/danger.ico")
}
