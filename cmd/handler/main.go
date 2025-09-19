package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// CANMessage is the generic structure we receive from the bus
type CANMessage struct {
	ID     string      `json:"id"`
	Length int         `json:"length"`
	Data   string      `json:"data"`
	Meta   interface{} `json:"meta"`
}

// BatteryState holds the decoded battery information
type BatteryState struct {
	HighCellVoltageID   int     `json:"high_cell_voltage_id"`
	HighCellVoltage     float64 `json:"high_cell_voltage_volts"`
	LowCellVoltageID    int     `json:"low_cell_voltage_id"`
	LowCellVoltage      float64 `json:"low_cell_voltage_volts"`
	CellDeltaMilliVolts int     `json:"cell_delta_mV"`
	DeltaVolts          float64 `json:"delta_volts"`
	PopulatedCells      int     `json:"populated_cells"`
	CRCOk               bool    `json:"crc_ok"`
}

// StateJSON is the overall JSON structure saved to disk
type StateJSON struct {
	Timestamp string       `json:"timestamp"`
	Battery   BatteryState `json:"battery"`
}

// CRC-8 SAE J1850 parameters (tentative)
// Polynomial: 0x1D, Init: 0xFF, XorOut: 0xFF
func crc8J1850(data []byte) byte {
	crc := byte(0xFF)
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x1D
			} else {
				crc <<= 1
			}
		}
	}
	return crc ^ 0xFF
}

func decode6B2(msg CANMessage) (*StateJSON, error) {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return nil, fmt.Errorf("invalid data length for 0x6B2: %d", len(msg.Data))
	}

	// Parse raw bytes from hex string
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		var b byte
		_, err := fmt.Sscanf(msg.Data[2*i:2*i+2], "%02X", &b)
		if err != nil {
			return nil, fmt.Errorf("invalid hex in data at byte %d: %v", i, err)
		}
		bytes[i] = b
	}

	// Mapping:
	// Byte0: reserved/unused
	// Byte1: High Cell ID (uint8)
	// Byte2-3: High Cell Voltage (big-endian, scale 1e-4 V)
	// Byte4: Low Cell ID (uint8)
	// Byte5: Delta from High to Low in millivolts (uint8, 1 LSB = 1 mV)
	// Byte6: Populated cells (uint8)
	// Byte7: CRC

	highID := int(bytes[1])
	highVoltRaw := (int(bytes[2]) << 8) | int(bytes[3])
	highVolt := float64(highVoltRaw) * 0.0001 // 1e-4 V per LSB

	lowID := int(bytes[4])
	delta_mV := int(bytes[5]) // 1 mV per LSB
	lowVolt := highVolt - float64(delta_mV)/1000.0
	if lowVolt < 0 {
		lowVolt = 0
	}
	if lowVolt > highVolt {
		lowVolt = highVolt
	}

	deltaVolts := highVolt - lowVolt

	popCells := int(bytes[6])
	crcByte := bytes[7]

	// CRC validation (tentative J1850)
	calc := crc8J1850(bytes[:7])
	crcOk := calc == crcByte

	state := &StateJSON{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Battery: BatteryState{
			HighCellVoltageID:   highID,
			HighCellVoltage:     highVolt,
			LowCellVoltageID:    lowID,
			LowCellVoltage:      lowVolt,
			CellDeltaMilliVolts: delta_mV,
			DeltaVolts:          deltaVolts,
			PopulatedCells:      popCells,
			CRCOk:               crcOk,
		},
	}

	return state, nil
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Drain()

	// Ensure data directory exists
	_ = os.MkdirAll("data", 0755)

	subject := "can.raw"
	_, err = nc.Subscribe(subject, func(m *nats.Msg) {
		var canMsg CANMessage
		if err := json.Unmarshal(m.Data, &canMsg); err != nil {
			log.Printf("Invalid JSON: %s", m.Data)
			return
		}

		if canMsg.ID == "6B2" {
			state, err := decode6B2(canMsg)
			if err != nil {
				log.Printf("Decode error: %v", err)
				return
			}

			// Save state.json (overwrite with latest)
			filePath := "data/state.json"
			f, err := os.Create(filePath)
			if err != nil {
				log.Printf("Failed to create state.json: %v", err)
				return
			}
			defer f.Close()

			enc := json.NewEncoder(f)
			enc.SetIndent("", "  ")
			if err := enc.Encode(state); err != nil {
				log.Printf("Failed to write state.json: %v", err)
			}

			log.Printf("Updated %s (CRC ok=%v).", filePath, state.Battery.CRCOk)
		}
	})

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Printf("Listening for CAN ID 6B2 on '%s'...", subject)
	select {}
}
