package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/finfinack/shellyExport/pkg/config"
	"github.com/finfinack/shellyExport/pkg/export"
	"github.com/finfinack/shellyExport/pkg/shelly"
)

var (
	configFile = flag.String("config", "config.json", "File path where the configuration is stored.")
)

const (
	urlPath = "/v2/statistics/power-consumption/em-3p"
)

func pullStatistics(cfg *config.Config) (*shelly.PowerConsumptionStatistics, error) {
	url, err := url.Parse(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server specified in config (%q): %s\n", cfg.Server, err)
	}
	url.Path = "/v2/statistics/power-consumption/em-3p"
	q := url.Query()
	q.Set("id", cfg.Devices[0].ID)
	q.Set("channel", "0")
	q.Set("date_range", "custom")
	q.Set("date_from", cfg.Timeframe.From.Format(shelly.DateTimeFmt))
	q.Set("date_to", cfg.Timeframe.To.Format(shelly.DateTimeFmt))
	q.Set("auth_key", cfg.AuthKey)
	url.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s\n", err)
	}
	if cfg.UserAgent != "" {
		req.Header.Add("User-Agent", cfg.UserAgent)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch data: %s\n", err)
	}
	if http.StatusOK != resp.StatusCode {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unable to fetch data (response code %d): %s", resp.StatusCode, b)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %s\n", err)
	}

	stats := &shelly.PowerConsumptionStatistics{}
	if err := json.Unmarshal(body, stats); err != nil {
		return nil, fmt.Errorf("unable to parse body as JSON: %s", err)
	}

	if stats.Interval != "day" {
		return nil, fmt.Errorf("returned interval is not supported: %q\n", stats.Interval)
	}

	return shelly.NormalizePowerConsumptionStatistics(stats, time.Time(cfg.Timeframe.From), time.Time(cfg.Timeframe.To)), nil
}

func main() {
	flag.Parse()

	cfg, err := config.ReadFromFile(*configFile)
	if err != nil {
		log.Fatalf("unable to read config: %s\n", err)
	}

	stats, err := pullStatistics(cfg)
	if err != nil {
		log.Fatalf("unable to pull statistics: %s", err)
	}

	export.ToCSV(stats, os.Stdout)
}
