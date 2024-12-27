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
	baseURLPath = "/v2/statistics/power-consumption"
	minInterval = 5 * 24 * time.Hour // 5 days
)

func pullStatisticsForTimeframe(cfg *config.Config, dev *config.Device, from, to time.Time) (*shelly.PowerConsumptionStatistics, error) {
	url, err := url.Parse(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server specified in config (%q): %s\n", cfg.Server, err)
	}
	devType := config.SupportedDeviceTypes[strings.ToLower(dev.Type)]
	url = url.JoinPath(url.Path, baseURLPath, devType.PathSuffix)
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

	switch devType.Phases {
	case 1:
		stats := &shelly.PowerConsumptionStatistics1p{}
		if err := json.Unmarshal(body, stats); err != nil {
			return nil, fmt.Errorf("unable to parse body as JSON: %s", err)
		}
		if stats.Interval != "day" {
			return nil, fmt.Errorf("returned interval is not supported: %q\n", stats.Interval)
		}
		stats.Normalize(from, to)
		return &shelly.PowerConsumptionStatistics{DeviceType: devType, Stats1p: stats}, nil
	case 3:
		stats := &shelly.PowerConsumptionStatistics3p{}
		if err := json.Unmarshal(body, stats); err != nil {
			return nil, fmt.Errorf("unable to parse body as JSON: %s", err)
		}
		if stats.Interval != "day" {
			return nil, fmt.Errorf("returned interval is not supported: %q\n", stats.Interval)
		}
		stats.Normalize(from, to)
		return &shelly.PowerConsumptionStatistics{DeviceType: devType, Stats3p: stats}, nil
	default:
		return nil, fmt.Errorf("unsupported amount of phases: %d", devType.Phases)
	}
}

func pullStatistics(cfg *config.Config, dev *config.Device) (*shelly.PowerConsumptionStatistics, error) {
	var stats *shelly.PowerConsumptionStatistics

	from := time.Time(cfg.Timeframe.From)
	to := time.Time(from).AddDate(0, 1, 0)
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

		switch statsFrame.DeviceType.Phases {
		case 1:
			if stats == nil {
				stats = statsFrame
			} else {
				stats.Stats1p.Add(statsFrame.Stats1p)
			}
		case 3:
			if stats == nil {
				stats = statsFrame
			} else {
				stats.Stats3p.Add(statsFrame.Stats3p)
			}
		default:
			return nil, fmt.Errorf("unsupported amount of phases: %d", statsFrame.DeviceType.Phases)
		}

		if !to.Before(time.Time(cfg.Timeframe.To)) {
			break
		}

		from = to
		to = time.Time(from).AddDate(0, 1, 0)
	}

	return stats, nil
}

func run(ctx context.Context, cfg *config.Config, outpfx string) error {
	for _, dev := range cfg.Devices {
		if dev.IsDisabled {
			log.Printf("skipping device %s (ID %s) because it is disabled\n", dev.Name, dev.ID)
			continue
		}

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
