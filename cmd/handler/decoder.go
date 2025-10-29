package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"
)

func decodeHexPayload(data string, expectedLen int) ([]byte, error) {
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex payload: %w", err)
	}
	if expectedLen > 0 && len(bytes) != expectedLen {
		return nil, fmt.Errorf("invalid data length: expected %d bytes, got %d", expectedLen, len(bytes))
	}
	return bytes, nil
}

func readUint16LE(b []byte, offset int) (uint16, error) {
	if offset+2 > len(b) {
		return 0, fmt.Errorf("offset %d out of range for %d byte slice", offset, len(b))
	}
	return binary.LittleEndian.Uint16(b[offset : offset+2]), nil
}

func readInt16LE(b []byte, offset int) (int16, error) {
	val, err := readUint16LE(b, offset)
	return int16(val), err
}

// decode6B1 handles High Cell ID, Pack Current, and High Cell Voltage
// 0x6B1 - 0015 FF92 9BF3 00ED
//
//	High Cell Id      -> 0015 = 21 -> 21
//	Pack Current      -> FF92 = -110 -> -11.0A (Int16 ÷ 10)
//	High Cell Voltage -> 9BF3 = 39,923 -> 3.9923 (/ 10000)
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

	// Parse pack current from bytes 2-3 (chars 4-7)
	packCurrentHex := msg.Data[4:8]
	packCurrentRaw, err := strconv.ParseUint(packCurrentHex, 16, 16)
	if err != nil {
		return fmt.Errorf("failed to parse pack current: %v", err)
	}

	// Parse High Cell Voltage from bytes 4-5 (chars 8-11)
	highCellVoltageHex := msg.Data[8:12]
	highCellVoltageRaw, err := strconv.ParseInt(highCellVoltageHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse high cell voltage: %v", err)
	}

	// Convert voltage: divide by 10000 to get correct value
	highCellVoltage := float64(highCellVoltageRaw) / 10000.0
	packCurrent := float64(int16(packCurrentRaw)) / 10.0

	// Update cell data
	cellData.HighCell.ID = int(highCellID)
	cellData.HighCell.Voltage = highCellVoltage
	cellData.PackData.PackCurrent = packCurrent
	timestamp := time.Now().Format(time.RFC3339Nano)
	cellData.LastUpdate.HighCell = timestamp
	cellData.LastUpdate.PackCurrent = timestamp

	// Write to JSON file
	if err := writeJSONFile(); err != nil {
		log.Printf("⚠️  Failed to write JSON: %v", err)
	}

	return nil
}

// decode6B2 handles Low Cell ID, Auxiliary System Voltage, and Low Cell Voltage
// 0x6B2 - 0046 0085 9BAA 0010
//
//	Low Cell Id      -> 0046 = 70 -> 70
//	12V System Volt. -> 0085 = 133 -> 13.3V (/ 10)
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

	// Parse 12V system voltage from bytes 2-3 (chars 4-7)
	auxVoltageHex := msg.Data[4:8]
	auxVoltageRaw, err := strconv.ParseInt(auxVoltageHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse auxiliary voltage: %v", err)
	}

	// Parse Low Cell Voltage from bytes 4-5 (chars 8-11)
	lowCellVoltageHex := msg.Data[8:12]
	lowCellVoltageRaw, err := strconv.ParseInt(lowCellVoltageHex, 16, 32)
	if err != nil {
		return fmt.Errorf("failed to parse low cell voltage: %v", err)
	}

	// Convert voltage: divide by 10000 to get correct value
	lowCellVoltage := float64(lowCellVoltageRaw) / 10000.0
	auxVoltage := float64(auxVoltageRaw) / 10.0

	// Update cell data
	cellData.LowCell.ID = int(lowCellID)
	cellData.LowCell.Voltage = lowCellVoltage
	cellData.AuxVoltage = auxVoltage
	cellData.LastUpdate.LowCell = time.Now().Format(time.RFC3339Nano)
	cellData.LastUpdate.AuxVoltage = cellData.LastUpdate.LowCell

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
	// The BMS reports CCL/DCL using 0.1A resolution; convert to amps.
	cellData.SystemControl.PackCCL = float64(packCCL) / 10.0
	cellData.SystemControl.PackDCL = float64(packDCL) / 10.0
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

// decodeBmsLimits handles CAN ID 0x351 (BmsLimits)
func decodeBmsLimits(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 8)
	if err != nil {
		return err
	}

	cv, err := readUint16LE(payload, 0)
	if err != nil {
		return err
	}
	cc, err := readUint16LE(payload, 2)
	if err != nil {
		return err
	}
	dc, err := readUint16LE(payload, 4)
	if err != nil {
		return err
	}
	dv, err := readUint16LE(payload, 6)
	if err != nil {
		return err
	}

	mainData.BmsLimits.ChargeVoltageLimit = float64(cv) / 10.0
	mainData.BmsLimits.ChargeCurrentLimit = float64(cc) / 10.0
	mainData.BmsLimits.DischargeCurrentLimit = float64(dc) / 10.0
	mainData.BmsLimits.DischargeVoltageLimit = float64(dv) / 10.0
	mainData.LastUpdate.BmsLimits = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.BmsLimits++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}

// decodeBmsSOC handles CAN ID 0x355 (BmsSOC)
func decodeBmsSOC(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 8)
	if err != nil {
		return err
	}

	socRaw, err := readUint16LE(payload, 0)
	if err != nil {
		return err
	}
	sohRaw, err := readUint16LE(payload, 2)
	if err != nil {
		return err
	}
	socHighRaw, err := readUint16LE(payload, 4)
	if err != nil {
		return err
	}

	mainData.BmsSOC.StateOfCharge = float64(socRaw)
	mainData.BmsSOC.StateOfHealth = float64(sohRaw)
	mainData.BmsSOC.StateOfChargeHighDef = float64(socHighRaw) / 10.0
	mainData.LastUpdate.BmsSOC = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.BmsSOC++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}

// decodeBmsStatus1 handles CAN ID 0x356 (BmsStatus1)
func decodeBmsStatus1(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 6)
	if err != nil {
		return err
	}

	packVoltRaw, err := readUint16LE(payload, 0)
	if err != nil {
		return err
	}
	packCurrentRaw, err := readInt16LE(payload, 2)
	if err != nil {
		return err
	}
	packTempRaw, err := readInt16LE(payload, 4)
	if err != nil {
		return err
	}

	mainData.BmsStatus1.PackVoltage = float64(packVoltRaw) / 10.0
	mainData.BmsStatus1.PackCurrent = float64(packCurrentRaw) / 10.0
	mainData.BmsStatus1.PackTemperature = float64(packTempRaw) / 10.0
	mainData.LastUpdate.BmsStatus1 = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.BmsStatus1++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}

// decodeBmsErrors handles CAN ID 0x35A (BmsErrors)
func decodeBmsErrors(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 4)
	if err != nil {
		return err
	}

	raw := binary.LittleEndian.Uint32(payload)

	mainData.BmsErrors.P0A06ChgLimitEnforceFault = (raw>>31)&1 == 1
	mainData.BmsErrors.P0A05InputPSUFault = (raw>>30)&1 == 1
	mainData.BmsErrors.P0AA6HVIsolationFault = (raw>>29)&1 == 1
	mainData.BmsErrors.P0560RedundantPSUFault = (raw>>28)&1 == 1
	mainData.BmsErrors.U0100ExternalComms = (raw>>27)&1 == 1
	mainData.BmsErrors.P0A9CThermistorFault = (raw>>26)&1 == 1
	mainData.BmsErrors.P0A81FanMonitorFault = (raw>>25)&1 == 1
	mainData.BmsErrors.P0A02WeakPackFault = (raw>>24)&1 == 1
	mainData.BmsErrors.P0A0FCellASICFault = (raw>>23)&1 == 1
	mainData.BmsErrors.P0A0DHighCell5VFault = (raw>>22)&1 == 1
	mainData.BmsErrors.P0AC0CurrentSensorFault = (raw>>21)&1 == 1
	mainData.BmsErrors.P0A04OpenWiringFault = (raw>>20)&1 == 1
	mainData.BmsErrors.P0AFALowCellVoltFault = (raw>>19)&1 == 1
	mainData.BmsErrors.P0A80WeakCellFault = (raw>>18)&1 == 1
	mainData.BmsErrors.P0A12CellBalanceOffFault = (raw>>17)&1 == 1
	mainData.BmsErrors.P0A1FInternalCommsFault = (raw>>16)&1 == 1
	mainData.BmsErrors.P0A10PackHotFault = (raw>>7)&1 == 1
	mainData.BmsErrors.P0A0ELowCellFault = (raw>>6)&1 == 1
	mainData.BmsErrors.P0A0CHighCellFault = (raw>>5)&1 == 1
	mainData.BmsErrors.P0A0BIntSWFault = (raw>>4)&1 == 1
	mainData.BmsErrors.P0A0AIntHeatsinkFault = (raw>>3)&1 == 1
	mainData.BmsErrors.P0A09InternalHWFault = (raw>>2)&1 == 1
	mainData.BmsErrors.P0A08ChgSafetyRelay = (raw>>1)&1 == 1
	mainData.BmsErrors.P0A07DischgLimitEnforce = raw&1 == 1
	mainData.LastUpdate.BmsErrors = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.BmsErrors++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}

// decodeBmsStatus2 handles CAN ID 0x35B (BmsStatus2)
func decodeBmsStatus2(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 4)
	if err != nil {
		return err
	}

	raw := binary.LittleEndian.Uint32(payload)
	isoRaw := raw & 0xFFFF

	mainData.BmsStatus2.IsolationADC = float64(isoRaw) * 0.001
	mainData.BmsStatus2.ReadyPower = (raw>>16)&1 == 1
	mainData.BmsStatus2.ChargePower = (raw>>17)&1 == 1
	mainData.BmsStatus2.DischargeRelay = (raw>>18)&1 == 1
	mainData.BmsStatus2.ChargeInterlock = (raw>>19)&1 == 1
	mainData.BmsStatus2.MPO1 = (raw>>20)&1 == 1
	mainData.BmsStatus2.MPO2 = (raw>>21)&1 == 1
	mainData.BmsStatus2.MPO3 = (raw>>22)&1 == 1
	mainData.BmsStatus2.MPO4 = (raw>>23)&1 == 1
	mainData.LastUpdate.BmsStatus2 = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.BmsStatus2++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}

// decodeDU1Feedback handles CAN ID 0x125 (DU1Feedback)
func decodeDU1Feedback(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 8)
	if err != nil {
		return err
	}

	dcRaw, err := readInt16LE(payload, 0)
	if err != nil {
		return err
	}
	voltageRaw, err := readUint16LE(payload, 2)
	if err != nil {
		return err
	}
	torqueRaw, err := readInt16LE(payload, 4)
	if err != nil {
		return err
	}
	acRaw, err := readInt16LE(payload, 6)
	if err != nil {
		return err
	}

	mainData.DU1Feedback.DCCurrent = float64(dcRaw) / 10.0
	mainData.DU1Feedback.BusVoltage = float64(voltageRaw) / 10.0
	mainData.DU1Feedback.ThrottleTorqueRequest = float64(torqueRaw)
	mainData.DU1Feedback.ACCurrent = float64(acRaw) / 10.0
	mainData.LastUpdate.DU1Feedback = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.DU1Feedback++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}

// decodeDU1Status handles CAN ID 0x126 (DU1Status)
func decodeDU1Status(msg CANMessage) error {
	payload, err := decodeHexPayload(msg.Data, 8)
	if err != nil {
		return err
	}

	// Lower nibble of byte 0 holds the op mode.
	mainData.DU1Status.OpMode = payload[0] & 0x0F

	// Lower nibble of byte 1 holds gear selection.
	mainData.DU1Status.Gear = payload[1] & 0x0F

	mainData.DU1Status.Mode = (payload[1] & 0x10) != 0
	mainData.DU1Status.DrivePowerLimited = (payload[1] & 0x20) != 0
	mainData.DU1Status.Error = (payload[1] & 0x40) != 0
	mainData.DU1Status.BrakeRegenLightRequest = (payload[1] & 0x80) != 0

	motorSpeedRaw, err := readInt16LE(payload, 2)
	if err != nil {
		return err
	}
	inverterTempRaw, err := readInt16LE(payload, 4)
	if err != nil {
		return err
	}
	motorTempRaw, err := readInt16LE(payload, 6)
	if err != nil {
		return err
	}

	mainData.DU1Status.MotorSpeed = float64(motorSpeedRaw)
	mainData.DU1Status.InverterTemp = float64(inverterTempRaw)
	mainData.DU1Status.MotorTemp = float64(motorTempRaw)
	mainData.LastUpdate.DU1Status = time.Now().Format(time.RFC3339Nano)
	mainData.MessageCount.DU1Status++

	if err := writeMainDataFile(); err != nil {
		log.Printf("⚠️  Failed to write main data: %v", err)
	}
	return nil
}
