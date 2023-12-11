package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func saveFlatStatistic(credentialsFilePath string, spreadsheetId string, dataRange string, t time.Time, data []flatStatItem) error {
	ctx := context.Background()

	b, err := os.ReadFile(credentialsFilePath)
	if err != nil {
		return fmt.Errorf("can't read credentials file: %w", err)
	}

	config, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return fmt.Errorf("can't read JWT config from json: %w", err)
	}
	client := config.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("can't create sheets service: %w", err)
	}

	var vr sheets.ValueRange

	for _, row := range data {
		value := []any{
			t.UTC().Format(time.DateTime),
			row.Category,
			row.Location,
			row.RoomsCount,
			row.MedianPrice,
		}

		vr.Values = append(vr.Values, value)
	}

	_, err = srv.Spreadsheets.Values.Append(spreadsheetId, dataRange, &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("can't write data to sheet: %w", err)
	}

	return nil
}
