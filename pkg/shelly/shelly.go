package shelly

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	DateTimeFmt = time.DateTime
)

type PowerConsumptionStatistics struct {
	Timezone string     `json:"timezone"`
	Interval string     `json:"interval"`
	History  [][]*Entry `json:"history"`
	Sum      []*Entry   `json:"sum"`
}

func (p *PowerConsumptionStatistics) Sort() {
	sort.Slice(p.History[0], func(i, j int) bool {
		return p.History[0][i].DateTime.Before(p.History[0][j].DateTime)
	})
	sort.Slice(p.History[1], func(i, j int) bool {
		return p.History[1][i].DateTime.Before(p.History[1][j].DateTime)
	})
	sort.Slice(p.History[2], func(i, j int) bool {
		return p.History[2][i].DateTime.Before(p.History[2][j].DateTime)
	})
	sort.Slice(p.Sum, func(i, j int) bool {
		return p.Sum[i].DateTime.Before(p.Sum[j].DateTime)
	})
}

func (p *PowerConsumptionStatistics) Add(stats *PowerConsumptionStatistics) error {
	if p.Timezone != stats.Timezone {
		return fmt.Errorf("timezone of this stats (%q) is different from the one to be added (%q)", p.Timezone, stats.Timezone)
	}
	if p.Interval != stats.Interval {
		return fmt.Errorf("interval of this stats (%q) is different from the one to be added (%q)", p.Interval, stats.Interval)
	}

	p.History[0] = append(p.History[0], stats.History[0]...)
	p.History[1] = append(p.History[1], stats.History[1]...)
	p.History[2] = append(p.History[2], stats.History[2]...)
	p.Sum = append(p.Sum, stats.Sum...)

	p.Sort()

	return nil
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

func NormalizePowerConsumptionStatistics(stats *PowerConsumptionStatistics, from, to time.Time) *PowerConsumptionStatistics {
	// Parse the existing data and normalize it by combining duplicate date/time entries.
	normalized := map[time.Time]map[string]*Entry{}
	date := from
	for date.Before(to) {
		for i := 0; i < len(stats.Sum); i++ {
			if stats.Sum[i].DateTime.Format(DateTimeFmt) != date.Format(DateTimeFmt) {
				continue
			}

			n, ok := normalized[date]
			if !ok {
				n = map[string]*Entry{}
			}

			// Phase A
			currentPhaseA := stats.History[0][i]
			pa, ok := n["phaseA"]
			if ok {
				n["phaseA"] = combineEntries(currentPhaseA, pa)
			} else {
				n["phaseA"] = currentPhaseA
			}

			// Phase B
			currentPhaseB := stats.History[1][i]
			pb, ok := n["phaseB"]
			if ok {
				n["phaseB"] = combineEntries(currentPhaseB, pb)
			} else {
				n["phaseB"] = currentPhaseB
			}

			// Phase C
			currentPhaseC := stats.History[2][i]
			pc, ok := n["phaseC"]
			if ok {
				n["phaseC"] = combineEntries(currentPhaseC, pc)
			} else {
				n["phaseC"] = currentPhaseC
			}

			// Total
			currentSum := stats.Sum[i]
			sum, ok := n["total"]
			if ok {
				n["total"] = combineEntries(currentSum, sum)
			} else {
				n["total"] = currentSum
			}

			normalized[date] = n
		}
		date = date.AddDate(0, 0, 1)
	}

	// Create the new structure and add the normalized data.
	out := &PowerConsumptionStatistics{
		Timezone: stats.Timezone,
		Interval: stats.Interval,
		History: [][]*Entry{
			{},
			{},
			{},
		},
		Sum: []*Entry{},
	}
	for _, entries := range normalized {
		out.History[0] = append(out.History[0], entries["phaseA"])
		out.History[1] = append(out.History[1], entries["phaseB"])
		out.History[2] = append(out.History[2], entries["phaseC"])
		out.Sum = append(out.Sum, entries["total"])
	}

	out.Sort()

	return out
}
