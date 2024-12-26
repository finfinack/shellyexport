package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	DateFmt = time.DateOnly
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

type Config struct {
	Timeframe *Timeframe `json:"timeframe"`
	UserAgent string     `json:"user_agent"`
	Server    string     `json:"server"`
	AuthKey   string     `json:"auth_key"`
	Devices   []*Device  `json:"devices"`
}

type Timeframe struct {
	From         ConfigDate `json:"from"`
	To           ConfigDate `json:"to"`
	LookbackDays int        `json:"lookback_days"`
}

type Device struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func Validate(config *Config) error {
	// Timeframe
	if config.Timeframe == nil {
		return errors.New("timeframe must be set")
	}
	if config.Timeframe.LookbackDays == 0 && (config.Timeframe.From.IsZero() || config.Timeframe.To.IsZero()) {
		return errors.New("either lookback_days or from and to need to be set")
	}
	if config.Timeframe.LookbackDays > 0 && (config.Timeframe.From.IsZero() || config.Timeframe.To.IsZero()) {
		return errors.New("either lookback_days or from and to need to be set")
	}
	if !config.Timeframe.From.IsZero() && config.Timeframe.To.IsZero() {
		return errors.New("when lookback_days is not set, both from and to need to be set")
	}

	// Device
	if len(config.Devices) != 1 {
		return errors.New("exactly one device needs to be set")
	}
	if config.Devices[0].ID == "" {
		return errors.New("device ID needs to be set")
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
