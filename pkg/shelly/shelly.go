package shelly

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/finfinack/shellyExport/pkg/config"
)

const (
	DateTimeFmt = time.DateTime
)

type PowerConsumptionStatistics struct {
	DeviceType *config.DeviceType
	Stats1p    *PowerConsumptionStatistics1p
	Stats3p    *PowerConsumptionStatistics3p
}

type Entry struct {
	IsMissing   bool       `json:"missing"`
	DateTime    ShellyTime `json:"datetime"`
	Consumption float64    `json:"consumption"`
	Channel     string     `json:"channel"`
	Reversed    float64    `json:"reversed"`
	MinVoltage  float64    `json:"min_voltage"`
	MaxVoltage  float64    `json:"max_voltage"`
	Purpose     string     `json:"purpose"`
	Cost        float64    `json:"cost"`
	TariffID    string     `json:"tariff_id"`
}

func combineEntries(entryA, entryB *Entry) *Entry {
	tariff := "multiple"
	if entryA.TariffID == entryB.TariffID {
		tariff = entryA.TariffID
	}
	purpose := "multiple"
	if entryA.Purpose == entryB.Purpose {
		purpose = entryA.Purpose
	}
	channel := "multiple"
	if entryA.Channel == entryB.Channel {
		channel = entryA.Channel
	}

	return &Entry{
		DateTime:    entryA.DateTime,
		Consumption: entryA.Consumption + entryB.Consumption,
		Channel:     channel,
		Reversed:    entryA.Reversed + entryB.Reversed,
		MinVoltage:  math.Min(entryA.MinVoltage, entryB.MinVoltage),
		MaxVoltage:  math.Max(entryA.MaxVoltage, entryB.MaxVoltage),
		Cost:        entryA.Cost + entryB.Cost,
		Purpose:     purpose,
		TariffID:    tariff,
	}
}

type ShellyTime time.Time

func (s *ShellyTime) UnmarshalJSON(b []byte) error {
	ts := strings.Trim(string(b), `"`)
	t, err := time.Parse(DateTimeFmt, ts)
	if err != nil {
		return err
	}
	*s = ShellyTime(t)
	return nil
}

func (s ShellyTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(s))
}

func (s ShellyTime) Format(ts string) string {
	return time.Time(s).Format(DateTimeFmt)
}

func (s ShellyTime) Before(t ShellyTime) bool {
	return time.Time(s).Before(time.Time(t))
}
