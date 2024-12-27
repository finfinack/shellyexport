package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/finfinack/shellyExport/pkg/shelly"
)

func ToCSV(stats *shelly.PowerConsumptionStatistics, w io.Writer) error {
	writer := csv.NewWriter(w)

	switch stats.DeviceType.Phases {
	case 1:
		writer.Write([]string{
			"day",
			"total",
			"total_returned",
			"is_missing",
		})
		for i := 0; i < len(stats.Stats1p.History); i++ {
			writer.Write([]string{
				stats.Stats1p.History[i].DateTime.Format(dayFmt),
				fmt.Sprintf("%f", stats.Stats1p.History[i].Consumption),
				fmt.Sprintf("%f", stats.Stats1p.History[i].Reversed),
				fmt.Sprintf("%t", stats.Stats1p.History[i].IsMissing),
			})
		}
	case 3:
		writer.Write([]string{
			"day",
			"phase_a",
			"phase_b",
			"phase_c",
			"total",
			"phase_a_returned",
			"phase_b_returned",
			"phase_c_returned",
			"total_returned",
			"is_missing",
		})
		for i := 0; i < len(stats.Stats3p.Sum); i++ {
			writer.Write([]string{
				stats.Stats3p.Sum[i].DateTime.Format(dayFmt),
				fmt.Sprintf("%f", stats.Stats3p.History[0][i].Consumption),
				fmt.Sprintf("%f", stats.Stats3p.History[1][i].Consumption),
				fmt.Sprintf("%f", stats.Stats3p.History[2][i].Consumption),
				fmt.Sprintf("%f", stats.Stats3p.Sum[i].Consumption),
				fmt.Sprintf("%f", stats.Stats3p.History[0][i].Reversed),
				fmt.Sprintf("%f", stats.Stats3p.History[1][i].Reversed),
				fmt.Sprintf("%f", stats.Stats3p.History[2][i].Reversed),
				fmt.Sprintf("%f", stats.Stats3p.Sum[i].Reversed),
				fmt.Sprintf("%t", stats.Stats3p.Sum[i].IsMissing),
			})
		}
	default:
		return fmt.Errorf("unsupported amount of phases: %d", stats.DeviceType.Phases)
	}

	writer.Flush()
	return nil
}
