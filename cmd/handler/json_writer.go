package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

// CellData represents individual cell information
type CellData struct {
	ID      int     `json:"id"`
	Voltage float64 `json:"voltage"`
}

// CellDataJSON is the structure saved to cell_data.json
type CellDataJSON struct {
	Timestamp  string   `json:"timestamp"`
	HighCell   CellData `json:"high_cell"`
	LowCell    CellData `json:"low_cell"`
	CellDelta  float64  `json:"cell_delta"`
	LastUpdate struct {
		HighCell string `json:"high_cell"`
		LowCell  string `json:"low_cell"`
	} `json:"last_update"`
}

var cellData *CellDataJSON

// writeJSONFile writes the current cell data to cell_data.json
func writeJSONFile() error {
	if cellData == nil {
		return fmt.Errorf("no cell data to write")
	}

	// Update timestamp and calculate delta (rounded to 4 decimals)
	cellData.Timestamp = time.Now().Format(time.RFC3339Nano)
	delta := cellData.HighCell.Voltage - cellData.LowCell.Voltage
	cellData.CellDelta = math.Round(delta*10000) / 10000

	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Write to file
	file, err := os.Create("data/cell_data.json")
	if err != nil {
		return fmt.Errorf("failed to create cell_data.json: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cellData); err != nil {
		return fmt.Errorf("failed to encode JSON: %v", err)
	}

	return nil
}

// initCellData initializes the cell data structure
func initCellData() {
	cellData = &CellDataJSON{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		HighCell:  CellData{ID: 0, Voltage: 0.0},
		LowCell:   CellData{ID: 0, Voltage: 0.0},
		CellDelta: 0.0,
	}
}
