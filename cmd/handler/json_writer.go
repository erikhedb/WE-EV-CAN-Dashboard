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

// PackData represents 6B0 battery pack status information
type PackData struct {
	SOC         float64 `json:"soc"`
	CellCount   int     `json:"cell_count"`
	PackVoltage float64 `json:"pack_voltage"`
	PackCurrent float64 `json:"pack_current"`
}

// RelayState represents the relay and system status bits
type RelayState struct {
	DischargeRelay bool `json:"discharge_relay"`
	ChargeRelay    bool `json:"charge_relay"`
	ChargerSafety  bool `json:"charger_safety"`
	MalfunctionDTC bool `json:"malfunction_dtc"`
	MPInput1       bool `json:"mp_input_1"`
	AlwaysOn       bool `json:"always_on"`
	IsReady        bool `json:"is_ready"`
	IsCharging     bool `json:"is_charging"`
	MPInput2       bool `json:"mp_input_2"`
	MPInput3       bool `json:"mp_input_3"`
	Reserved       bool `json:"reserved"`
	MPOutput2      bool `json:"mp_output_2"`
	MPOutput3      bool `json:"mp_output_3"`
	MPOutput4      bool `json:"mp_output_4"`
	MPEnable       bool `json:"mp_enable"`
	MPOutput1      bool `json:"mp_output_1"`
}

// TemperatureData represents 6B3 temperature information
type TemperatureData struct {
	HighTemp int `json:"high_temp"`
	LowTemp  int `json:"low_temp"`
}

// SystemControl represents 6B4 system control data
type SystemControl struct {
	RelayState RelayState `json:"relay_state"`
	PackCCL    float64    `json:"pack_ccl"`
	PackDCL    float64    `json:"pack_dcl"`
}

// CellDataJSON is the structure saved to ev_data.json
type CellDataJSON struct {
	Timestamp       string          `json:"timestamp"`
	PackData        PackData        `json:"pack_data"`
	HighCell        CellData        `json:"high_cell"`
	LowCell         CellData        `json:"low_cell"`
	AuxVoltage      float64         `json:"aux_voltage"`
	CellDelta       float64         `json:"cell_delta"`
	TemperatureData TemperatureData `json:"temperature_data"`
	SystemControl   SystemControl   `json:"system_control"`
	LastUpdate      struct {
		PackData        string `json:"pack_data"`
		PackCurrent     string `json:"pack_current"`
		HighCell        string `json:"high_cell"`
		LowCell         string `json:"low_cell"`
		AuxVoltage      string `json:"aux_voltage"`
		TemperatureData string `json:"temperature_data"`
		SystemControl   string `json:"system_control"`
	} `json:"last_update"`
}

var cellData *CellDataJSON

// writeJSONFile writes the current cell data to ev_data.json
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
	file, err := os.Create("data/ev_data.json")
	if err != nil {
		return fmt.Errorf("failed to create ev_data.json: %v", err)
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
		PackData: PackData{
			SOC:         0.0,
			CellCount:   0,
			PackVoltage: 0.0,
			PackCurrent: 0.0,
		},
		HighCell:   CellData{ID: 0, Voltage: 0.0},
		LowCell:    CellData{ID: 0, Voltage: 0.0},
		AuxVoltage: 0.0,
		CellDelta:  0.0,
		TemperatureData: TemperatureData{
			HighTemp: 0,
			LowTemp:  0,
		},
		SystemControl: SystemControl{
			RelayState: RelayState{},
			PackCCL:    0,
			PackDCL:    0,
		},
	}
}
