package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"website-checker/internal/app"
	"website-checker/internal/config"
	"website-checker/internal/notification"
	"website-checker/internal/systray"
)

// Глобальные переменные для управления
var (
	cfg          *config.Config
)

func main() {
	currentDir, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	currentDir = filepath.Dir(currentDir)
	defaultConfig := filepath.Join(currentDir, "config.yml")
	configFile := flag.String("config", defaultConfig, "Путь к файлу конфигурации")

	flag.Parse()

	// Загрузка конфигурации
	cfg, err = config.Load(*configFile)
	if err != nil {
		notification.Error("Ошибка загрузки конфигурации", err.Error())
		os.Exit(1)
	}

	// Инициализация уведомлений
	notification.Init(app.AppName)
	notification.SendConfigLoaded(*cfg)

	// Запуск в режиме с иконкой в трее
	systray.Run(cfg, *configFile)
}
