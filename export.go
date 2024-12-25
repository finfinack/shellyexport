package main

import (
	"encoding/json"
	"flag"
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
	configFile   = flag.String("config", "config.json", "File path where the configuration is stored.")
	lookbackDays = flag.Int("lookback", 30, "Number of past days to fetch data for.")
)

const (
	urlPath = "/v2/statistics/power-consumption/em-3p"
)

func main() {
	flag.Parse()

	cfg, err := config.ReadFromFile(*configFile)
	if err != nil {
		log.Fatalf("unable to read config: %s\n", err)
	}
	if len(cfg.Devices) != 1 {
		log.Fatalln("config needs to contain exactly one device at the moment")
	}

	from := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -*lookbackDays)
	to := time.Now().UTC().Truncate(24 * time.Hour)

	url, err := url.Parse(cfg.Server)
	if err != nil {
		log.Fatalf("invalid server specified in config (%q): %s\n", cfg.Server, err)
	}
	url.Path = "/v2/statistics/power-consumption/em-3p"
	q := url.Query()
	q.Set("id", cfg.Devices[0].ID)
	q.Set("channel", "0")
	q.Set("date_range", "custom")
	q.Set("date_from", from.Format(shelly.DateTimeFmt))
	q.Set("date_to", to.Format(shelly.DateTimeFmt))
	q.Set("auth_key", cfg.AuthKey)
	url.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		log.Fatalf("unable to create request: %s\n", err)
	}
	if cfg.UserAgent != "" {
		req.Header.Add("User-Agent", cfg.UserAgent)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("unable to fetch data: %s\n", err)
	}
	if http.StatusOK != resp.StatusCode {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("unable to fetch data (response code %d): %s", resp.StatusCode, b)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("unable to read response body: %s\n", err)
	}

	stats := &shelly.PowerConsumptionStatistics{}
	if err := json.Unmarshal(body, stats); err != nil {
		log.Fatalf("unable to parse body as JSON: %s", err)
	}

	if stats.Interval != "day" {
		log.Fatalf("returned interval is not supported: %q\n", stats.Interval)
	}

	stats = shelly.NormalizePowerConsumptionStatistics(stats, from, to)
	export.ToCSV(stats, os.Stdout)
}
