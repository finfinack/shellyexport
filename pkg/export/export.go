package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/finfinack/shellyExport/pkg/shelly"
)

const (
	dayFmt = time.DateOnly
)

func ToCSV(stats *shelly.PowerConsumptionStatistics, w io.Writer) {
	writer := csv.NewWriter(w)
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

	for i := 0; i < len(stats.Sum); i++ {
		writer.Write([]string{
			stats.Sum[i].DateTime.Format(dayFmt),
			fmt.Sprintf("%f", stats.History[0][i].Consumption),
			fmt.Sprintf("%f", stats.History[1][i].Consumption),
			fmt.Sprintf("%f", stats.History[2][i].Consumption),
			fmt.Sprintf("%f", stats.Sum[i].Consumption),
			fmt.Sprintf("%f", stats.History[0][i].Reversed),
			fmt.Sprintf("%f", stats.History[1][i].Reversed),
			fmt.Sprintf("%f", stats.History[2][i].Reversed),
			fmt.Sprintf("%f", stats.Sum[i].Reversed),
			fmt.Sprintf("%t", stats.Sum[i].IsMissing),
		})
	}
	writer.Flush()
}
