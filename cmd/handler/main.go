package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// CANMessage is the generic structure we receive from the bus
type CANMessage struct {
	ID     string      `json:"id"`
	Length int         `json:"length"`
	Data   string      `json:"data"`
	Meta   interface{} `json:"meta"`
}

func main() {
	// Initialize cell data structure
	initCellData()

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Drain()

	subject := "can.raw"
	_, err = nc.Subscribe(subject, func(m *nats.Msg) {
		var canMsg CANMessage
		if err := json.Unmarshal(m.Data, &canMsg); err != nil {
			log.Printf("Invalid JSON: %s", string(m.Data))
			return
		}

		// Filter for only 6B0, 6B1, 6B2, 6B3, and 6B4 messages
		switch canMsg.ID {
		case "6B0":
			if err := decode6B0(canMsg); err != nil {
				log.Printf("Error decoding 6B0: %v", err)
			}
		case "6B1":
			if err := decode6B1(canMsg); err != nil {
				log.Printf("Error decoding 6B1: %v", err)
			}
		case "6B2":
			if err := decode6B2(canMsg); err != nil {
				log.Printf("Error decoding 6B2: %v", err)
			}
		case "6B3":
			if err := decode6B3(canMsg); err != nil {
				log.Printf("Error decoding 6B3: %v", err)
			}
		case "6B4":
			if err := decode6B4(canMsg); err != nil {
				log.Printf("Error decoding 6B4: %v", err)
			}
		default:
			// Ignore all other messages
			return
		}
	})

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Printf("CAN Handler started - listening on '%s'", subject)
	log.Println("Filtering for CAN IDs: 6B0 (Pack Status), 6B1 (High Cell), 6B2 (Low Cell), 6B3 (Temperature), 6B4 (System Control)")
	log.Println("Decoded pack status, cell voltage, temperature, and system control data will be written to JSON")

	// Keep the program running
	select {}
}
