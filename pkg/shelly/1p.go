package shelly

import (
	"fmt"
	"sort"
	"time"
)

type PowerConsumptionStatistics1p struct {
	Timezone string   `json:"timezone"`
	Interval string   `json:"interval"`
	History  []*Entry `json:"history"`
}

func (p *PowerConsumptionStatistics1p) GetInterval() string {
	return p.Interval
}

func (p *PowerConsumptionStatistics1p) Sort() {
	sort.Slice(p.History, func(i, j int) bool {
		return p.History[i].DateTime.Before(p.History[j].DateTime)
	})
}

func (p *PowerConsumptionStatistics1p) Add(stats *PowerConsumptionStatistics1p) error {
	if p.Timezone != stats.Timezone {
		return fmt.Errorf("timezone of this stats (%q) is different from the one to be added (%q)", p.Timezone, stats.Timezone)
	}
	if p.Interval != stats.Interval {
		return fmt.Errorf("interval of this stats (%q) is different from the one to be added (%q)", p.Interval, stats.Interval)
	}

	p.History = append(p.History, stats.History...)
	p.Sort()
	return nil
}

func (p *PowerConsumptionStatistics1p) Normalize(from, to time.Time) {
	// Parse the existing data and normalize it by combining duplicate date/time entries.
	normalized := map[time.Time]*Entry{}
	date := from
	for date.Before(to) {
		for i := 0; i < len(p.History); i++ {
			if p.History[i].DateTime.Format(DateTimeFmt) != date.Format(DateTimeFmt) {
				continue
			}

			current := p.History[i]
			n, ok := normalized[date]
			if ok {
				n = combineEntries(current, n)
			} else {
				n = current
			}

			normalized[date] = n
		}
		date = date.AddDate(0, 0, 1)
	}

	// Create the new structure and add the normalized data.
	out := &PowerConsumptionStatistics1p{
		Timezone: p.Timezone,
		Interval: p.Interval,
		History:  []*Entry{},
	}
	for _, entry := range normalized {
		out.History = append(out.History, entry)
	}
	out.Sort()
	p = out
}
