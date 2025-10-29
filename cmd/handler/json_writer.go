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

// BmsLimitsData represents the decoded contents of CAN ID 0x351.
type BmsLimitsData struct {
	ChargeVoltageLimit    float64 `json:"charge_voltage_limit"`
	ChargeCurrentLimit    float64 `json:"charge_current_limit"`
	DischargeCurrentLimit float64 `json:"discharge_current_limit"`
	DischargeVoltageLimit float64 `json:"discharge_voltage_limit"`
}

// BmsSOCData represents the decoded contents of CAN ID 0x355.
type BmsSOCData struct {
	StateOfCharge        float64 `json:"state_of_charge"`
	StateOfHealth        float64 `json:"state_of_health"`
	StateOfChargeHighDef float64 `json:"state_of_charge_high_def"`
}

// BmsStatus1Data holds pack voltage, current, and temperature info from CAN ID 0x356.
type BmsStatus1Data struct {
	PackVoltage     float64 `json:"pack_voltage"`
	PackCurrent     float64 `json:"pack_current"`
	PackTemperature float64 `json:"pack_temperature"`
}

// BmsErrorsData represents the bit-field diagnostics from CAN ID 0x35A.
type BmsErrorsData struct {
	P0A06ChgLimitEnforceFault bool `json:"p0a06_chg_limit_enforce_fault"`
	P0A05InputPSUFault        bool `json:"p0a05_input_psu_fault"`
	P0AA6HVIsolationFault     bool `json:"p0aa6_hv_isolation_fault"`
	P0560RedundantPSUFault    bool `json:"p0560_redundant_psu_fault"`
	U0100ExternalComms        bool `json:"u0100_external_comms"`
	P0A9CThermistorFault      bool `json:"p0a9c_thermistor_fault"`
	P0A81FanMonitorFault      bool `json:"p0a81_fan_monitor_fault"`
	P0A02WeakPackFault        bool `json:"p0a02_weak_pack_fault"`
	P0A0FCellASICFault        bool `json:"p0a0f_cell_asic_fault"`
	P0A0DHighCell5VFault      bool `json:"p0a0d_high_cell_5v_fault"`
	P0AC0CurrentSensorFault   bool `json:"p0ac0_current_sensor_fault"`
	P0A04OpenWiringFault      bool `json:"p0a04_open_wiring_fault"`
	P0AFALowCellVoltFault     bool `json:"p0afa_low_cell_volt_fault"`
	P0A80WeakCellFault        bool `json:"p0a80_weak_cell_fault"`
	P0A12CellBalanceOffFault  bool `json:"p0a12_cell_balance_off_fault"`
	P0A1FInternalCommsFault   bool `json:"p0a1f_internal_comms_fault"`
	P0A10PackHotFault         bool `json:"p0a10_pack_hot_fault"`
	P0A0ELowCellFault         bool `json:"p0a0e_low_cell_fault"`
	P0A0CHighCellFault        bool `json:"p0a0c_high_cell_fault"`
	P0A0BIntSWFault           bool `json:"p0a0b_int_sw_fault"`
	P0A0AIntHeatsinkFault     bool `json:"p0a0a_int_heatsink_fault"`
	P0A09InternalHWFault      bool `json:"p0a09_internal_hw_fault"`
	P0A08ChgSafetyRelay       bool `json:"p0a08_chg_safety_relay"`
	P0A07DischgLimitEnforce   bool `json:"p0a07_dischg_limit_enforce_fault"`
}

// BmsStatus2Data represents contactor/relay states from CAN ID 0x35B.
type BmsStatus2Data struct {
	MPO4            bool    `json:"mpo4"`
	MPO3            bool    `json:"mpo3"`
	MPO2            bool    `json:"mpo2"`
	MPO1            bool    `json:"mpo1"`
	ChargeInterlock bool    `json:"charge_interlock"`
	DischargeRelay  bool    `json:"discharge_relay"`
	ChargePower     bool    `json:"charge_power"`
	ReadyPower      bool    `json:"ready_power"`
	IsolationADC    float64 `json:"isolation_adc_volts"`
}

// DU1FeedbackData represents CAN ID 0x125 telemetry.
type DU1FeedbackData struct {
	DCCurrent             float64 `json:"dc_current"`
	BusVoltage            float64 `json:"bus_voltage"`
	ThrottleTorqueRequest float64 `json:"throttle_torque_request"`
	ACCurrent             float64 `json:"ac_current"`
}

// DU1StatusData represents CAN ID 0x126 state information.
type DU1StatusData struct {
	BrakeRegenLightRequest bool    `json:"brake_regen_light_request"`
	Error                  bool    `json:"error"`
	DrivePowerLimited      bool    `json:"drive_power_limited"`
	Mode                   bool    `json:"mode"`
	MotorTemp              float64 `json:"motor_temp"`
	InverterTemp           float64 `json:"inverter_temp"`
	MotorSpeed             float64 `json:"motor_speed"`
	Gear                   uint8   `json:"gear"`
	OpMode                 uint8   `json:"op_mode"`
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

// MainDataJSON aggregates key BMS/drive-unit messages into main_data.json.
type MainDataJSON struct {
	Timestamp    string          `json:"timestamp"`
	BmsLimits    BmsLimitsData   `json:"bms_limits"`
	BmsSOC       BmsSOCData      `json:"bms_soc"`
	BmsStatus1   BmsStatus1Data  `json:"bms_status_1"`
	BmsErrors    BmsErrorsData   `json:"bms_errors"`
	BmsStatus2   BmsStatus2Data  `json:"bms_status_2"`
	DU1Feedback  DU1FeedbackData `json:"du1_feedback"`
	DU1Status    DU1StatusData   `json:"du1_status"`
	MessageCount struct {
		BmsLimits   int `json:"bms_limits"`
		BmsSOC      int `json:"bms_soc"`
		BmsStatus1  int `json:"bms_status_1"`
		BmsErrors   int `json:"bms_errors"`
		BmsStatus2  int `json:"bms_status_2"`
		DU1Feedback int `json:"du1_feedback"`
		DU1Status   int `json:"du1_status"`
	} `json:"message_count"`
	LastUpdate struct {
		BmsLimits   string `json:"bms_limits"`
		BmsSOC      string `json:"bms_soc"`
		BmsStatus1  string `json:"bms_status_1"`
		BmsErrors   string `json:"bms_errors"`
		BmsStatus2  string `json:"bms_status_2"`
		DU1Feedback string `json:"du1_feedback"`
		DU1Status   string `json:"du1_status"`
	} `json:"last_update"`
}

var (
	cellData *CellDataJSON
	mainData *MainDataJSON
)

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

// writeMainDataFile writes the aggregated BMS/drive-unit data to main_data.json.
func writeMainDataFile() error {
	if mainData == nil {
		return fmt.Errorf("no main data to write")
	}

	mainData.Timestamp = time.Now().Format(time.RFC3339Nano)

	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	file, err := os.Create("data/main_data.json")
	if err != nil {
		return fmt.Errorf("failed to create main_data.json: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(mainData); err != nil {
		return fmt.Errorf("failed to encode main data JSON: %v", err)
	}

	return nil
}

// initMainData initializes the main data structure.
func initMainData() {
	mainData = &MainDataJSON{
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}
}
