package export

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/finfinack/shellyExport/pkg/config"
	"github.com/finfinack/shellyExport/pkg/shelly"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	valueInputOption = "USER_ENTERED" // https://developers.google.com/sheets/api/reference/rest/v4/ValueInputOption
	insertDataOption = "INSERT_ROWS"  // https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/append#InsertDataOption
)

func ToGoogleSheet(ctx context.Context, stats *shelly.PowerConsumptionStatistics, cfg *config.GoogleSheet) error {
	creds, err := base64.StdEncoding.DecodeString(cfg.SvcAcctKey)
	if err != nil {
		return fmt.Errorf("unable to decode service account key: %s", err)
	}

	config, err := google.JWTConfigFromJSON(creds, sheets.SpreadsheetsScope)
	if err != nil {
		return fmt.Errorf("unable to create JWT config from JSON: %s", err)
	}

	client := config.Client(ctx)
	svc, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to create new service: %s", err)
	}

	// TODO: Check if header needs to be written.
	values := &sheets.ValueRange{
		Values: [][]interface{}{
			{
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
			},
		},
	}
	// TODO:
	// - check which days already exist and overwrite if necessary (instead of appending)
	// - write date as date without empty time
	// - skip / overwrite missing values at the end in particular
	for i := 0; i < len(stats.Sum); i++ {
		values.Values = append(values.Values, []interface{}{
			stats.Sum[i].DateTime.Format(dayFmt),
			stats.History[0][i].Consumption,
			stats.History[1][i].Consumption,
			stats.History[2][i].Consumption,
			stats.Sum[i].Consumption,
			stats.History[0][i].Reversed,
			stats.History[1][i].Reversed,
			stats.History[2][i].Reversed,
			stats.Sum[i].Reversed,
			stats.Sum[i].IsMissing,
		})
	}

	resp, err := svc.Spreadsheets.Values.Append(cfg.SpreadsheetID, cfg.SheetID, values).ValueInputOption(valueInputOption).InsertDataOption(insertDataOption).Context(ctx).Do()
	if err != nil || resp.HTTPStatusCode != 200 {
		return err
	}

	return nil
}
