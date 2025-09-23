package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

const writeInterval = time.Second // Timer interval for writing JSON to file

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

// BMSState holds decoded values from 0x6B0
type BMSState struct {
	PackCurrent    float64 `json:"pack_current_amps"`
	PackVoltage    float64 `json:"pack_voltage_volts"`
	StateOfCharge  float64 `json:"soc_percent"`
	RelayState     uint16  `json:"relay_state_raw"`
	RelayMain      bool    `json:"relay_main"`
	RelayCharge    bool    `json:"relay_charge"`
	RelayDischarge bool    `json:"relay_discharge"`
	CRCChecksum    byte    `json:"crc_checksum"`
}

// StateJSON is the overall JSON structure saved to disk
type StateJSON struct {
	Timestamp string       `json:"timestamp"`
	Battery   BatteryState `json:"battery"`
	BMS       BMSState     `json:"bms"`
}

var globalState *StateJSON

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

func decode6B2(msg CANMessage) (*BatteryState, error) {
	if len(msg.Data) != 16 { // 8 bytes = 16 hex chars
		return nil, fmt.Errorf("invalid data length for 0x6B2: %d", len(msg.Data))
	}

	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		var b byte
		_, err := fmt.Sscanf(msg.Data[2*i:2*i+2], "%02X", &b)
		if err != nil {
			return nil, fmt.Errorf("invalid hex in data at byte %d: %v", i, err)
		}
		bytes[i] = b
	}

	highID := int(bytes[1])
	highVoltRaw := (int(bytes[2]) << 8) | int(bytes[3])
	highVolt := float64(highVoltRaw) * 0.0001

	lowID := int(bytes[4])
	delta_mV := int(bytes[5])
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

	calc := crc8J1850(bytes[:7])
	crcOk := calc == crcByte

	return &BatteryState{
		HighCellVoltageID:   highID,
		HighCellVoltage:     highVolt,
		LowCellVoltageID:    lowID,
		LowCellVoltage:      lowVolt,
		CellDeltaMilliVolts: delta_mV,
		DeltaVolts:          deltaVolts,
		PopulatedCells:      popCells,
		CRCOk:               crcOk,
	}, nil
}

func decode6B0(msg CANMessage) (*BMSState, error) {
	if len(msg.Data) != 16 {
		return nil, fmt.Errorf("invalid data length for 0x6B0: %d", len(msg.Data))
	}

	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		var b byte
		_, err := fmt.Sscanf(msg.Data[2*i:2*i+2], "%02X", &b)
		if err != nil {
			return nil, fmt.Errorf("invalid hex in data at byte %d: %v", i, err)
		}
		bytes[i] = b
	}

	// Pack current from bytes1-2
	packCurrentRaw := (uint16(bytes[1]) << 8) | uint16(bytes[2])
	packCurrent := float64(packCurrentRaw) * 0.1

	// Pack voltage from bytes3-4, scale 0.125 V/LSB
	packVoltRaw := (uint16(bytes[3]) << 8) | uint16(bytes[4])
	packVolt := float64(packVoltRaw) * 0.125

	// SOC from byte4
	socRaw := bytes[4]
	soc := float64(socRaw) * 0.5

	// Relay state from bytes6-7
	relayState := (uint16(bytes[6]) << 8) | uint16(bytes[7])

	crc := bytes[7]

	return &BMSState{
		PackCurrent:    packCurrent,
		PackVoltage:    packVolt,
		StateOfCharge:  soc,
		RelayState:     relayState,
		RelayMain:      relayState&0x01 != 0,
		RelayCharge:    relayState&0x02 != 0,
		RelayDischarge: relayState&0x04 != 0,
		CRCChecksum:    crc,
	}, nil
}

func initState() *StateJSON {
	return &StateJSON{
		Timestamp: "",
		Battery:   BatteryState{},
		BMS:       BMSState{},
	}
}

func writeStatePeriodically() {
	ticker := time.NewTicker(writeInterval)
	defer ticker.Stop()
	for range ticker.C {
		if globalState == nil {
			continue
		}
		globalState.Timestamp = time.Now().Format(time.RFC3339Nano)
		filePath := "data/state.json"
		f, err := os.Create(filePath)
		if err != nil {
			log.Printf("Failed to create state.json: %v", err)
			continue
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(globalState); err != nil {
			log.Printf("Failed to write state.json: %v", err)
		}
		f.Close()
		log.Printf("Periodic update %s.", filePath)
	}
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Drain()

	_ = os.MkdirAll("data", 0755)

	globalState = initState()
	go writeStatePeriodically()

	subject := "can.raw"
	_, err = nc.Subscribe(subject, func(m *nats.Msg) {
		var canMsg CANMessage
		if err := json.Unmarshal(m.Data, &canMsg); err != nil {
			log.Printf("Invalid JSON: %s", m.Data)
			return
		}

		switch canMsg.ID {
		case "6B2":
			battery, err := decode6B2(canMsg)
			if err != nil {
				log.Printf("Decode error 6B2: %v", err)
				return
			}
			globalState.Battery = *battery
		case "6B0":
			bms, err := decode6B0(canMsg)
			if err != nil {
				log.Printf("Decode error 6B0: %v", err)
				return
			}
			globalState.BMS = *bms
		}
	})

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Printf("Listening for CAN IDs 6B2 and 6B0 on '%s'...", subject)
	select {}
}
