package main

import (
	"os"
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
	// Загрузка конфигурации
	cfg, configFilePath, err := config.Load()
	if err != nil {
		notification.Error("Ошибка загрузки конфигурации", err.Error())
		os.Exit(1)
	}
	// Инициализация уведомлений
	notification.Init(app.AppName)
	notification.SendConfigLoaded(*cfg)

	// Запуск в режиме с иконкой в трее
	systray.Run(cfg, *configFilePath)
}
