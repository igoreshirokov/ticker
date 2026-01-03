package main

import (
	"os"
	"website-checker/internal/app"
	"website-checker/internal/config"
	"website-checker/internal/i18n"
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
		// Переводим сообщение об ошибке
		notification.Error("Configuration file does not exist: " + err.Error())
		os.Exit(1)
	}

	i18n.Load(cfg.General.Lang)
	app.AppName = i18n.T("app_name")

	// Инициализация уведомлений
	notification.Init(app.AppName, cfg)
	notification.SendConfigLoaded()

	// Запуск в режиме с иконкой в трее
	systray.Run(cfg, *configFilePath)
}
