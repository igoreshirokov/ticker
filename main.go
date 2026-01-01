package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gen2brain/beeep"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sites          []SiteConfig `yaml:"sites"`
	Notifications  Notifications `yaml:"notifications"`
	General        GeneralConfig `yaml:"general"`
}

type SiteConfig struct {
	URL     string `yaml:"url"`
	Name    string `yaml:"name"`
	Timeout int    `yaml:"timeout"`
}

type Notifications struct {
	ShowPopup    bool `yaml:"show_popup"`
	ConsoleOutput bool `yaml:"console_output"`
}

type GeneralConfig struct {
	CheckInterval   int `yaml:"check_interval"`
	ConcurrentChecks int `yaml:"concurrent_checks"`
}

type CheckResult struct {
	Site      SiteConfig
	Success   bool
	StatusCode int
	Error     string
	Duration  time.Duration
}

func main() {
	configFile := flag.String("config", "config.yml", "Путь к файлу конфигурации")
	once := flag.Bool("once", false, "Выполнить только одну проверку и выйти")
	verbose := flag.Bool("v", false, "Подробный вывод")

	flag.Parse()

	config, err := loadConfig(*configFile)
	if err != nil {
		fmt.Printf("Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Загружено %d сайтов для проверки\n", len(config.Sites))
	fmt.Printf("Интервал проверки: %d секунд\n", config.General.CheckInterval)

	if *once {
		fmt.Println("Запуск проверки сайтов только один раз...")
	}

	if *verbose {
		fmt.Println("Включение подробного вывода...")
	}

	results := checkAllSites(config, *verbose)
	for _, result := range results {
		fmt.Printf("Результат проверки %s: %v (статус код: %d, время: %v)\n", result.Site.Name, result.Success, result.StatusCode, result.Duration)
	}

	// Проверка, все ли сайты работают
	allOK := allSitesOK(results)
	if allOK {
		fmt.Println("🎉 Все сайты работают нормально!")
		sendSuccessNotification(config)
	} else {
		fmt.Println("⚠️ Обнаружены проблемы с некоторыми сайтами")
		sendFailNotification(config, results)
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

	beeep.AppName = "Ticker"
	title := "Website Checker"
	msg := "✅ Все сайты работают нормально!"
	iconPath := "assets/info.png"
	// Используем beeep для уведомлений Windows
	err := beeep.Notify(
		title,
		msg,
		iconPath, // Можно заменить на свой иконку
	)
	
	if err != nil {
		fmt.Printf("Ошибка отправки уведомления: %v\n", err)
	}
}

func sendFailNotification(config *Config, failedResults []CheckResult) {
    if !config.Notifications.ShowPopup || len(failedResults) == 0 {
        return
    }

    beeep.AppName = "Ticker"
    title := "Website Checker"
    
    // Формируем сообщение с деталями
    msg := "⚠️ Обнаружены проблемы с сайтами:\n\n"
    for _, result := range failedResults {
        statusText := "ОШИБКА"
        if result.StatusCode > 0 {
            statusText = fmt.Sprintf("Статус: %d", result.StatusCode)
        }
        
        duration := result.Duration.Round(time.Millisecond)
        msg += fmt.Sprintf("• %s: %s (время: %v)\n", 
            result.Site.Name, 
            statusText, 
            duration,
        )
    }

    iconPath := "assets/danger.png"
    err := beeep.Notify(title, msg, iconPath)
    
    if err != nil {
        fmt.Printf("Ошибка отправки уведомления: %v\n", err)
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
			
			// Ограничиваем количество одновременных запросов
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
	
	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: time.Duration(site.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	// Создаем запрос
	req, err := http.NewRequest("GET", site.URL, nil)
	if err != nil {
		return CheckResult{
			Site:     site,
			Success:  false,
			Error:    fmt.Sprintf("Ошибка создания запроса: %v", err),
			Duration: time.Since(start),
		}
	}

	// Устанавливаем User-Agent
	req.Header.Set("User-Agent", "WebsiteChecker/1.0")

	// Выполняем запрос
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

	// Читаем немного тела ответа (чтобы убедиться, что соединение действительно работает)
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

	// Проверяем статус код
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