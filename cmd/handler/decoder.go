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
