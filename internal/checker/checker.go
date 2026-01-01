package checker

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"website-checker/internal/config"
)

type CheckResult struct {
	Site       config.SiteConfig
	Success    bool
	StatusCode int
	Error      string
	Duration   time.Duration
}


func CheckSite(site *config.SiteConfig, verbose bool) CheckResult {
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
			Site:     *site,
			Success:  false,
			Error:    fmt.Sprintf("Ошибка создания запроса: %v", err),
			Duration: time.Since(start),
		}
	}

	req.Header.Set("User-Agent", "WebsiteChecker/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			Site:     *site,
			Success:  false,
			Error:    fmt.Sprintf("Ошибка соединения: %v", err),
			Duration: time.Since(start),
		}
	}
	defer resp.Body.Close()

	_, err = io.CopyN(io.Discard, resp.Body, 4096)
	if err != nil && err != io.EOF {
		return CheckResult{
			Site:       *site,
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
		Site:       *site,
		Success:    success,
		StatusCode: resp.StatusCode,
		Error:      "",
		Duration:   time.Since(start),
	}
}

func CheckAllSites(configuration *config.Config, verbose bool) []CheckResult {
	var wg sync.WaitGroup
	results := make([]CheckResult, len(configuration.Sites))
	semaphore := make(chan struct{}, configuration.General.ConcurrentChecks)

	for i, site := range configuration.Sites {
		wg.Add(1)
		go func(idx int, site config.SiteConfig) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results[idx] = CheckSite(&site, verbose)
		}(i, site)
	}

	wg.Wait()
	return results
}
