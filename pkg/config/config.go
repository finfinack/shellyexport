package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	UserAgent string    `json:"user_agent"`
	Server    string    `json:"server"`
	AuthKey   string    `json:"auth_key"`
	Devices   []*Device `json:"devices"`
}

type Device struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
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

	return config, nil
}
