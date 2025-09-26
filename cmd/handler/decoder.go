package main

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// decode6B1 handles High Cell ID and Voltage
// 0x6B1 - 0012 0000 993A 009E
//
//	High Cell Id     -> 0012 = 18 -> 18
//	High Cell Voltage -> 993A = 39,226 -> 3.9336 (/ 10000)
func decode6B1(msg CANMessage) error {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return fmt.Errorf("invalid data length for 6B1: expected 16, got %d", len(msg.Data))
	}

	// Parse High Cell ID from bytes 0-1 (first 4 hex chars)
	highCellIDHex := msg.Data[0:4]
	highCellID, err := strconv.ParseInt(highCellIDHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse high cell ID: %v", err)
	}

	// Parse High Cell Voltage from bytes 4-5 (chars 8-11)
	highCellVoltageHex := msg.Data[8:12]
	highCellVoltageRaw, err := strconv.ParseInt(highCellVoltageHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse high cell voltage: %v", err)
	}

	// Convert voltage: divide by 10000 to get correct value
	highCellVoltage := float64(highCellVoltageRaw) / 10000.0

	// Update cell data
	cellData.HighCell.ID = int(highCellID)
	cellData.HighCell.Voltage = highCellVoltage
	cellData.LastUpdate.HighCell = time.Now().Format(time.RFC3339Nano)

	// Write to JSON file
	if err := writeJSONFile(); err != nil {
		log.Printf("⚠️  Failed to write JSON: %v", err)
	}

	return nil
}

// decode6B2 handles Low Cell ID and Voltage
// 0x6B2 - 0046 0000 9901 00E0
//
//	Low Cell Id      -> 0046 = 70 -> 70
//	Low Cell Voltage -> 9901 = 39.169 -> 3.9169 (/ 10000)
func decode6B2(msg CANMessage) error {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return fmt.Errorf("invalid data length for 6B2: expected 16, got %d", len(msg.Data))
	}

	// Parse Low Cell ID from bytes 0-1 (first 4 hex chars)
	lowCellIDHex := msg.Data[0:4]
	lowCellID, err := strconv.ParseInt(lowCellIDHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse low cell ID: %v", err)
	}

	// Parse Low Cell Voltage from bytes 4-5 (chars 8-11)
	lowCellVoltageHex := msg.Data[8:12]
	lowCellVoltageRaw, err := strconv.ParseInt(lowCellVoltageHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse low cell voltage: %v", err)
	}

	// Convert voltage: divide by 10000 to get correct value
	lowCellVoltage := float64(lowCellVoltageRaw) / 10000.0

	// Update cell data
	cellData.LowCell.ID = int(lowCellID)
	cellData.LowCell.Voltage = lowCellVoltage
	cellData.LastUpdate.LowCell = time.Now().Format(time.RFC3339Nano)

	// Write to JSON file
	if err := writeJSONFile(); err != nil {
		log.Printf("⚠️  Failed to write JSON: %v", err)
	}

	return nil
}

// decode6B4 handles System Control Information
// 0x6B4 - 0162 0004 1200 0000
//
//	Relay State -> 0162 = 0000 0001 0110 0010 (16-bit relay status)
//	Pack CCL    -> 0004 = 4A (Charge Current Limit)
//	Pack DCL    -> 1200 = 4608 (Discharge Current Limit - needs verification)
func decode6B4(msg CANMessage) error {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return fmt.Errorf("invalid data length for 6B4: expected 16, got %d", len(msg.Data))
	}

	// Parse Relay State from bytes 0-1 (first 4 hex chars) - 16-bit value
	relayStateHex := msg.Data[0:4]
	relayStateRaw, err := strconv.ParseInt(relayStateHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse relay state: %v", err)
	}

	// Parse Pack CCL from bytes 2-3 (chars 4-7)
	packCCLHex := msg.Data[4:8]
	packCCL, err := strconv.ParseInt(packCCLHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse pack CCL: %v", err)
	}

	// Parse Pack DCL from bytes 4-5 (chars 8-11)
	packDCLHex := msg.Data[8:12]
	packDCL, err := strconv.ParseInt(packDCLHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse pack DCL: %v", err)
	}

	// Decode relay state bits
	relayState := RelayState{
		DischargeRelay: (relayStateRaw & 0x01) != 0,   // Bit 0
		ChargeRelay:    (relayStateRaw & 0x02) != 0,   // Bit 1
		ChargerSafety:  (relayStateRaw & 0x04) != 0,   // Bit 2
		MalfunctionDTC: (relayStateRaw & 0x08) != 0,   // Bit 3
		MPInput1:       (relayStateRaw & 0x10) != 0,   // Bit 4
		AlwaysOn:       (relayStateRaw & 0x20) != 0,   // Bit 5
		IsReady:        (relayStateRaw & 0x40) != 0,   // Bit 6
		IsCharging:     (relayStateRaw & 0x80) != 0,   // Bit 7
		MPInput2:       (relayStateRaw & 0x0100) != 0, // Bit 8
		MPInput3:       (relayStateRaw & 0x0200) != 0, // Bit 9
		Reserved:       (relayStateRaw & 0x0400) != 0, // Bit 10
		MPOutput2:      (relayStateRaw & 0x0800) != 0, // Bit 11
		MPOutput3:      (relayStateRaw & 0x1000) != 0, // Bit 12
		MPOutput4:      (relayStateRaw & 0x2000) != 0, // Bit 13
		MPEnable:       (relayStateRaw & 0x4000) != 0, // Bit 14
		MPOutput1:      (relayStateRaw & 0x8000) != 0, // Bit 15
	}

	// Update cell data
	cellData.SystemControl.RelayState = relayState
	cellData.SystemControl.PackCCL = int(packCCL)
	cellData.SystemControl.PackDCL = int(packDCL)
	cellData.LastUpdate.SystemControl = time.Now().Format(time.RFC3339Nano)

	// Write to JSON file
	if err := writeJSONFile(); err != nil {
		log.Printf("⚠️  Failed to write JSON: %v", err)
	}

	return nil
}

// decode6B3 handles Temperature Information
// 0x6B3 - 0013 0000 0010 0000
//
//	High Temp -> 0013 = 19°C
//	Low Temp  -> 0010 = 16°C
func decode6B3(msg CANMessage) error {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return fmt.Errorf("invalid data length for 6B3: expected 16, got %d", len(msg.Data))
	}

	// Parse High Temperature from bytes 0-1 (first 4 hex chars)
	highTempHex := msg.Data[0:4]
	highTemp, err := strconv.ParseInt(highTempHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse high temperature: %v", err)
	}

	// Parse Low Temperature from bytes 4-5 (chars 8-11)
	lowTempHex := msg.Data[8:12]
	lowTemp, err := strconv.ParseInt(lowTempHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse low temperature: %v", err)
	}

	// Update temperature data
	cellData.TemperatureData.HighTemp = int(highTemp)
	cellData.TemperatureData.LowTemp = int(lowTemp)
	cellData.LastUpdate.TemperatureData = time.Now().Format(time.RFC3339Nano)

	// Write to JSON file
	if err := writeJSONFile(); err != nil {
		log.Printf("⚠️  Failed to write JSON: %v", err)
	}

	return nil
}

// decode6B0 handles Battery Pack Status
// 0x6B0 - 00A100486E50005F
//
//	SOC          -> 00A1 = 161 -> 80.5% (161 × 10 ÷ 2)
//	Cell Count   -> 0048 = 72 -> 72 (direct)
//	Pack Voltage -> 6E50 = 28,240 -> 282.4V (28240 ÷ 100)
func decode6B0(msg CANMessage) error {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return fmt.Errorf("invalid data length for 6B0: expected 16, got %d", len(msg.Data))
	}

	// Parse SOC from bytes 0-1 (first 4 hex chars)
	socHex := msg.Data[0:4]
	socRaw, err := strconv.ParseInt(socHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse SOC: %v", err)
	}

	// Parse Cell Count from bytes 2-3 (chars 4-7)
	cellCountHex := msg.Data[4:8]
	cellCount, err := strconv.ParseInt(cellCountHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse cell count: %v", err)
	}

	// Parse Pack Voltage from bytes 4-5 (chars 8-11)
	packVoltageHex := msg.Data[8:12]
	packVoltageRaw, err := strconv.ParseInt(packVoltageHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse pack voltage: %v", err)
	}

	// Convert values according to documentation
	soc := float64(socRaw*10) / 2.0                // 161 × 10 ÷ 2 = 80.5%
	packVoltage := float64(packVoltageRaw) / 100.0 // 28240 ÷ 100 = 282.4V

	// Update pack data
	cellData.PackData.SOC = soc
	cellData.PackData.CellCount = int(cellCount)
	cellData.PackData.PackVoltage = packVoltage
	cellData.LastUpdate.PackData = time.Now().Format(time.RFC3339Nano)

	// Write to JSON file
	if err := writeJSONFile(); err != nil {
		log.Printf("⚠️  Failed to write JSON: %v", err)
	}

	return nil
}
