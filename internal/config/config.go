package config

import (
	"fmt"
	"os"
	"time"

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

func Load(filename string) (*Config, error) {
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