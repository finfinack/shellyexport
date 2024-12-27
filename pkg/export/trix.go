package export

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/finfinack/shellyExport/pkg/config"
	"github.com/finfinack/shellyExport/pkg/shelly"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	valueInputOptionUserEntered = "USER_ENTERED" // https://developers.google.com/sheets/api/reference/rest/v4/ValueInputOption
	insertDataOptionInsertRows  = "INSERT_ROWS"  // https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/append#InsertDataOption
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

	switch stats.DeviceType.Phases {
	case 1:
		return trixExport1p(ctx, cfg, svc, stats.Stats1p)
	case 3:
		return trixExport3p(ctx, cfg, svc, stats.Stats3p)
	default:
		return fmt.Errorf("unsupported amount of phases: %d", stats.DeviceType.Phases)
	}
}

func trixExport1p(ctx context.Context, cfg *config.GoogleSheet, svc *sheets.Service, stats *shelly.PowerConsumptionStatistics1p) error {
	// Overwrite headers
	values := &sheets.ValueRange{
		Values: [][]interface{}{
			{
				"date",
				"total",
				"total_returned",
				"is_missing",
			},
		},
	}
	hdrResp, err := svc.Spreadsheets.Values.Update(cfg.SpreadsheetID, fmt.Sprintf("%s!A1:J1", cfg.SheetID), values).ValueInputOption(valueInputOptionUserEntered).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update header values: %s", err)
	}
	if hdrResp.HTTPStatusCode != 200 {
		return fmt.Errorf("unable to update header values: HTTP code %d", hdrResp.HTTPStatusCode)
	}

	// Figure out whether there is overlap.
	firstDate := time.Time(stats.History[0].DateTime)
	getResp, err := svc.Spreadsheets.Values.Get(cfg.SpreadsheetID, fmt.Sprintf("%s!A:A", cfg.SheetID)).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update header values: %s", err)
	}
	if hdrResp.HTTPStatusCode != 200 {
		return fmt.Errorf("unable to update header values: HTTP code %d", hdrResp.HTTPStatusCode)
	}
	rowIdx := 1
	for _, row := range getResp.Values {
		if rowIdx == 1 {
			rowIdx += 1
			continue // skip header
		}
		if row[0].(string) == "" {
			break
		}
		rowDate, err := time.Parse(time.DateTime, row[0].(string))
		if err != nil {
			rowDate, err = time.Parse(dayFmt, row[0].(string))
			if err != nil {
				return fmt.Errorf("unable to parse row as date/time: %s", row[0])
			}
		}
		if rowDate.Before(firstDate) {
			rowIdx += 1
			continue
		}
		break
	}

	values = &sheets.ValueRange{Values: [][]interface{}{}}
	for i := 0; i < len(stats.History); i++ {
		values.Values = append(values.Values, []interface{}{
			stats.History[i].DateTime.Format(dayFmt),
			stats.History[i].Consumption,
			stats.History[i].Reversed,
			stats.History[i].IsMissing,
		})
	}

	upResp, err := svc.Spreadsheets.Values.Update(cfg.SpreadsheetID, fmt.Sprintf("%s!A%d:J%d", cfg.SheetID, rowIdx, rowIdx+len(stats.History)), values).ValueInputOption(valueInputOptionUserEntered).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update values: %s", err)
	}
	if upResp.HTTPStatusCode != 200 {
		return fmt.Errorf("unable to update values: HTTP code %d", upResp.HTTPStatusCode)
	}

	return nil
}

func trixExport3p(ctx context.Context, cfg *config.GoogleSheet, svc *sheets.Service, stats *shelly.PowerConsumptionStatistics3p) error {
	// Overwrite headers
	values := &sheets.ValueRange{
		Values: [][]interface{}{
			{
				"date",
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
	hdrResp, err := svc.Spreadsheets.Values.Update(cfg.SpreadsheetID, fmt.Sprintf("%s!A1:J1", cfg.SheetID), values).ValueInputOption(valueInputOptionUserEntered).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update header values: %s", err)
	}
	if hdrResp.HTTPStatusCode != 200 {
		return fmt.Errorf("unable to update header values: HTTP code %d", hdrResp.HTTPStatusCode)
	}

	// Figure out whether there is overlap.
	firstDate := time.Time(stats.Sum[0].DateTime)
	getResp, err := svc.Spreadsheets.Values.Get(cfg.SpreadsheetID, fmt.Sprintf("%s!A:A", cfg.SheetID)).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update header values: %s", err)
	}
	if hdrResp.HTTPStatusCode != 200 {
		return fmt.Errorf("unable to update header values: HTTP code %d", hdrResp.HTTPStatusCode)
	}
	rowIdx := 1
	for _, row := range getResp.Values {
		if rowIdx == 1 {
			rowIdx += 1
			continue // skip header
		}
		if row[0].(string) == "" {
			break
		}
		rowDate, err := time.Parse(time.DateTime, row[0].(string))
		if err != nil {
			rowDate, err = time.Parse(dayFmt, row[0].(string))
			if err != nil {
				return fmt.Errorf("unable to parse row as date/time: %s", row[0])
			}
		}
		if rowDate.Before(firstDate) {
			rowIdx += 1
			continue
		}
		break
	}

	values = &sheets.ValueRange{Values: [][]interface{}{}}
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

	upResp, err := svc.Spreadsheets.Values.Update(cfg.SpreadsheetID, fmt.Sprintf("%s!A%d:J%d", cfg.SheetID, rowIdx, rowIdx+len(stats.Sum)), values).ValueInputOption(valueInputOptionUserEntered).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update values: %s", err)
	}
	if upResp.HTTPStatusCode != 200 {
		return fmt.Errorf("unable to update values: HTTP code %d", upResp.HTTPStatusCode)
	}

	return nil
}
