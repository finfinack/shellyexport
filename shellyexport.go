package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/finfinack/shellyExport/pkg/config"
	"github.com/finfinack/shellyExport/pkg/export"
	"github.com/finfinack/shellyExport/pkg/shelly"
)

var (
	configFile = flag.String("config", "config.json", "File path where the configuration is stored.")
	outfilePfx = flag.String("out", "", "File path used as a prefix for where the output is written to.")
)

const (
	urlPath     = "/v2/statistics/power-consumption/em-3p"
	minInterval = 5 * 24 * time.Hour // 5 days
)

func pullStatisticsForTimeframe(cfg *config.Config, dev *config.Device, from, to time.Time) (*shelly.PowerConsumptionStatistics, error) {
	url, err := url.Parse(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server specified in config (%q): %s\n", cfg.Server, err)
	}
	url.Path = "/v2/statistics/power-consumption/em-3p"
	q := url.Query()
	q.Set("id", dev.ID)
	q.Set("channel", "0")
	q.Set("date_range", "custom")
	q.Set("date_from", from.Format(shelly.DateTimeFmt))
	q.Set("date_to", to.Format(shelly.DateTimeFmt))
	q.Set("auth_key", cfg.AuthKey)
	url.RawQuery = q.Encode()

	log.Printf("requesting stats for device %q (ID %s) from %q to %q\n", dev.Name, dev.ID, from.Format(shelly.DateTimeFmt), to.Format(shelly.DateTimeFmt))

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

	return shelly.NormalizePowerConsumptionStatistics(stats, from, to), nil
}

func pullStatistics(cfg *config.Config, dev *config.Device) (*shelly.PowerConsumptionStatistics, error) {
	var stats *shelly.PowerConsumptionStatistics

	from := time.Time(cfg.Timeframe.From)
	to := time.Time(cfg.Timeframe.From).AddDate(0, 1, 0)
	for {
		if !to.Before(time.Time(cfg.Timeframe.To)) {
			to = time.Time(cfg.Timeframe.To)
		}
		// In some cases the above may lead to the interval being switched to "hour" instead
		// of "day" and we do not support that (yet). Instead we rather risk getting data
		// for future dates.
		if to.Sub(from) < minInterval {
			to = from.Add(minInterval)
		}

		statsFrame, err := pullStatisticsForTimeframe(cfg, dev, from, to)
		if err != nil {
			return nil, fmt.Errorf("unable to pull statistics from %q to %q: %s", from, to, err)
		}
		if stats == nil {
			stats = statsFrame
		} else {
			stats.Add(statsFrame)
		}

		if !to.Before(time.Time(cfg.Timeframe.To)) {
			break
		}

		from = to
		to = time.Time(cfg.Timeframe.From).AddDate(0, 1, 0)
	}

	return stats, nil
}

func run(ctx context.Context, cfg *config.Config, outpfx string) error {
	for _, dev := range cfg.Devices {
		stats, err := pullStatistics(cfg, dev)
		if err != nil {
			return fmt.Errorf("unable to pull statistics: %s", err)
		}

		var out io.Writer
		if dev.GoogleSheet == nil {
			out = os.Stdout
		}
		if outpfx != "" {
			name := dev.Name
			if name == "" {
				name = dev.ID
			}
			name = strings.ReplaceAll(strings.ToLower(name), " ", "_")
			outfile := fmt.Sprintf("%s-dev-%s.csv", outpfx, name)
			out, err = os.Create(outfile)
			if err != nil {
				return fmt.Errorf("unable to open file %q for writing: %s", outfile, err)
			} else {
				log.Printf("writing output for device %q (ID %q) to %q\n", dev.Name, dev.ID, outfile)
			}
		}
		if out != nil {
			export.ToCSV(stats, out)
		}

		if dev.GoogleSheet != nil {
			if err := export.ToGoogleSheet(ctx, stats, dev.GoogleSheet); err != nil {
				return fmt.Errorf("unable to export to sheet: %s", err)
			}
		}
	}

	return nil
}

func main() {
	flag.Parse()
	ctx := context.Background()

	cfg, err := config.ReadFromFile(*configFile)
	if err != nil {
		log.Fatalf("unable to read config: %s", err)
	}

	if err := run(ctx, cfg, *outfilePfx); err != nil {
		log.Fatal(err)
	}
}
