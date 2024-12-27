package shelly

import (
	"fmt"
	"sort"
	"time"
)

type PowerConsumptionStatistics3p struct {
	Timezone string     `json:"timezone"`
	Interval string     `json:"interval"`
	History  [][]*Entry `json:"history"`
	Sum      []*Entry   `json:"sum"`
}

func (p *PowerConsumptionStatistics3p) GetInterval() string {
	return p.Interval
}

func (p *PowerConsumptionStatistics3p) Sort() {
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

func (p *PowerConsumptionStatistics3p) Add(stats *PowerConsumptionStatistics3p) error {
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

func (p *PowerConsumptionStatistics3p) Normalize(from, to time.Time) {
	// Parse the existing data and normalize it by combining duplicate date/time entries.
	normalized := map[time.Time]map[string]*Entry{}
	date := from
	for date.Before(to) {
		for i := 0; i < len(p.Sum); i++ {
			if p.Sum[i].DateTime.Format(DateTimeFmt) != date.Format(DateTimeFmt) {
				continue
			}

			n, ok := normalized[date]
			if !ok {
				n = map[string]*Entry{}
			}

			// Phase A
			currentPhaseA := p.History[0][i]
			pa, ok := n["phaseA"]
			if ok {
				n["phaseA"] = combineEntries(currentPhaseA, pa)
			} else {
				n["phaseA"] = currentPhaseA
			}

			// Phase B
			currentPhaseB := p.History[1][i]
			pb, ok := n["phaseB"]
			if ok {
				n["phaseB"] = combineEntries(currentPhaseB, pb)
			} else {
				n["phaseB"] = currentPhaseB
			}

			// Phase C
			currentPhaseC := p.History[2][i]
			pc, ok := n["phaseC"]
			if ok {
				n["phaseC"] = combineEntries(currentPhaseC, pc)
			} else {
				n["phaseC"] = currentPhaseC
			}

			// Total
			currentSum := p.Sum[i]
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
	history := [][]*Entry{{}, {}, {}}
	sum := []*Entry{}
	for _, entries := range normalized {
		history[0] = append(history[0], entries["phaseA"])
		history[1] = append(history[1], entries["phaseB"])
		history[2] = append(history[2], entries["phaseC"])
		sum = append(sum, entries["total"])
	}
	p.History = history
	p.Sum = sum
	p.Sort()
}
