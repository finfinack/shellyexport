package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	DateFmt = time.DateOnly
)

var (
	supportedDeviceTypes = []string{
		"em-3p",
	}
)

type ConfigDate time.Time

func (d *ConfigDate) UnmarshalJSON(b []byte) error {
	ts := strings.Trim(string(b), `"`)
	t, err := time.Parse(DateFmt, ts)
	if err != nil {
		return err
	}
	*d = ConfigDate(t)
	return nil
}

func (d ConfigDate) IsZero() bool {
	return time.Time(d).IsZero()
}

func (d ConfigDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d))
}

func (d ConfigDate) Format(ts string) string {
	return time.Time(d).Format(DateFmt)
}

func (d ConfigDate) Before(e ConfigDate) bool {
	return time.Time(d).Before(time.Time(e))
}

type Config struct {
	Timeframe   *Timeframe   `json:"timeframe"`
	UserAgent   string       `json:"user_agent"`
	Server      string       `json:"server"`
	AuthKey     string       `json:"auth_key"`
	Devices     []*Device    `json:"devices"`
	GoogleSheet *GoogleSheet `json:"google_sheet"`
}

type Timeframe struct {
	From         ConfigDate `json:"from"`
	To           ConfigDate `json:"to"`
	LookbackDays int        `json:"lookback_days"`
}

type Device struct {
	ID          string       `json:"id"`
	Name        string       `json:"name,omitempty"`
	Type        string       `json:"type"`
	IsDisabled  bool         `json:"disabled"`
	GoogleSheet *GoogleSheet `json:"google_sheet"`
}

type GoogleSheet struct {
	SvcAcctKey    string `json:"service_account_key"`
	SheetID       string `json:"sheet_id"`
	SpreadsheetID string `json:"spreadsheet_id"`
}

func Validate(config *Config) error {
	// Timeframe
	if config.Timeframe == nil {
		return errors.New("timeframe must be set")
	}
	if config.Timeframe.LookbackDays == 0 && (config.Timeframe.From.IsZero() || config.Timeframe.To.IsZero()) {
		return errors.New("either lookback_days or from and to need to be set")
	}
	if config.Timeframe.LookbackDays > 0 && (!config.Timeframe.From.IsZero() || !config.Timeframe.To.IsZero()) {
		return errors.New("when lookbac_days is set, both from and to cannot be set")
	}
	if !config.Timeframe.From.IsZero() && config.Timeframe.To.IsZero() {
		return errors.New("when lookback_days is not set, both from and to need to be set")
	}
	if config.Timeframe.To.Before(config.Timeframe.From) {
		return errors.New("from date needs to be before to date")
	}

	// Google Sheet
	if config.GoogleSheet != nil {
		if config.GoogleSheet.SvcAcctKey == "" {
			return errors.New("svc_acct_key must be set for Google Sheet exports")
		}
	}

	// Device
	if len(config.Devices) == 0 {
		return errors.New("at least one device needs to be set")
	}
	for i, dev := range config.Devices {
		if dev.ID == "" {
			return fmt.Errorf("device ID needs to be set for device %d", i)
		}
		if !slices.Contains(supportedDeviceTypes, strings.ToLower(dev.Type)) {
			return fmt.Errorf("device type %q is not supported: %s", dev.Type, supportedDeviceTypes)
		}
		if dev.GoogleSheet != nil {
			if dev.GoogleSheet.SheetID == "" && config.GoogleSheet != nil {
				dev.GoogleSheet.SheetID = config.GoogleSheet.SheetID
			}
			if dev.GoogleSheet.SheetID == "" {
				return fmt.Errorf("sheet_id must be set for device %d (or globally)", i)
			}
			if dev.GoogleSheet.SpreadsheetID == "" && config.GoogleSheet != nil {
				dev.GoogleSheet.SpreadsheetID = config.GoogleSheet.SpreadsheetID
			}
			if dev.GoogleSheet.SpreadsheetID == "" {
				return fmt.Errorf("spreadsheet_id must be set for device %d (or globally)", i)
			}
			if dev.GoogleSheet.SvcAcctKey == "" && config.GoogleSheet != nil {
				dev.GoogleSheet.SvcAcctKey = config.GoogleSheet.SvcAcctKey
			}
			if dev.GoogleSheet.SvcAcctKey == "" {
				return fmt.Errorf("service_account_key must be set for device %d (or globally)", i)
			}
		}
	}

	// Auth
	if config.Server == "" {
		return errors.New("server needs to be set")
	}
	if config.AuthKey == "" {
		return errors.New("auth key needs to be set")
	}

	return nil
}

func ReadFromFile(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read config from %q: %s", file, err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("unable to parse JSON in %q: %s", file, err)
	}

	if err := Validate(config); err != nil {
		return nil, err
	}

	if config.Timeframe.LookbackDays > 0 {
		config.Timeframe.From = ConfigDate(time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -config.Timeframe.LookbackDays))
		config.Timeframe.To = ConfigDate(time.Now().UTC().Truncate(24 * time.Hour))
	}

	return config, nil
}
