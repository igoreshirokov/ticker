package notification

import (
	"fmt"
	"time"
	"website-checker/internal/app"
	"website-checker/internal/checker"
	"website-checker/internal/config"
	"website-checker/internal/i18n"

	"github.com/gen2brain/beeep"
)

var cfg *config.Config

func Init(appName string, globalConfig *config.Config) {
	beeep.AppName = appName
	cfg = globalConfig
}

func SendSuccess() {
	if !cfg.Notifications.ShowPopup {
		return
	}
	title := i18n.T("success_title")
	msg := i18n.T("success_msg")

	beeep.Notify(title, msg, app.IconGood)
}

func SendFail(failedResults []checker.CheckResult) {
	if !cfg.Notifications.ShowPopup || len(failedResults) == 0 {
		return
	}
	
	title := i18n.T("fail_title")
	msg := ""
	for _, result := range failedResults {
		
		duration := result.Duration.Round(time.Millisecond)
		msg += fmt.Sprintf("• %s: %s (время: %v)\n",
			result.Site.Name, result.Error, duration)
	}
	msg = i18n.T("fail_msg", "sites", msg)
	beeep.Alert(title, msg, app.IconBad)
}

func SendConfigLoaded() {
	if !cfg.Notifications.ShowPopup {
		return
	}
	title := i18n.T("config_load_title")
	msg := i18n.T("config_load_msg", "count", len(cfg.Sites))

	beeep.Notify(title, msg, app.IconGood)
}

func ShowLog(lastCheckResult string) {
	if !cfg.Notifications.ShowPopup {
		return
	}
	title := i18n.T("log_title")
	beeep.Alert(title, lastCheckResult, "")
}

func Error(msg string) {
	title := i18n.T("error_title")
	beeep.Alert(title, msg, app.IconBad)
}