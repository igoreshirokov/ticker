package checker

import (
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"time"

	"website-checker/internal/config"
	"website-checker/internal/i18n"
)

type CheckResult struct {
	Site       config.SiteConfig
	Success    bool
	StatusCode int
	Error      string
	Duration   time.Duration
}


func CheckSite(site *config.SiteConfig) CheckResult {
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
			Error:    i18n.T("error_create_request", err),

			Duration: time.Since(start),
		}
	}

	req.Header.Set("User-Agent", "WebsiteChecker/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			Site:     *site,
			Success:  false,
			Error:    i18n.T("error_connection", err),
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
			Error:      i18n.T("error_read_response", err),
			Duration:   time.Since(start),
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300


	return CheckResult{
		Site:       *site,
		Success:    success,
		StatusCode: resp.StatusCode,
		Error:      "",
		Duration:   time.Since(start),
	}
}

func CheckAllSites(configuration *config.Config) []CheckResult {
	var wg sync.WaitGroup
	results := make([]CheckResult, len(configuration.Sites))
	semaphore := make(chan struct{}, configuration.General.ConcurrentChecks)

	for i, site := range configuration.Sites {
		wg.Add(1)
		go func(idx int, site config.SiteConfig) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results[idx] = CheckSite(&site)
		}(i, site)
	}

	wg.Wait()
	return results
}
