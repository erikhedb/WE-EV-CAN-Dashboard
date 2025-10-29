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
	initMainData()

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

		// Filter for tracked CAN IDs that we decode into JSON
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
		case "351":
			if err := decodeBmsLimits(canMsg); err != nil {
				log.Printf("Error decoding 351: %v", err)
			}
		case "355":
			if err := decodeBmsSOC(canMsg); err != nil {
				log.Printf("Error decoding 355: %v", err)
			}
		case "356":
			if err := decodeBmsStatus1(canMsg); err != nil {
				log.Printf("Error decoding 356: %v", err)
			}
		case "35A":
			if err := decodeBmsErrors(canMsg); err != nil {
				log.Printf("Error decoding 35A: %v", err)
			}
		case "35B":
			if err := decodeBmsStatus2(canMsg); err != nil {
				log.Printf("Error decoding 35B: %v", err)
			}
		case "125":
			if err := decodeDU1Feedback(canMsg); err != nil {
				log.Printf("Error decoding 125: %v", err)
			}
		case "126":
			if err := decodeDU1Status(canMsg); err != nil {
				log.Printf("Error decoding 126: %v", err)
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
	log.Println("Additional IDs captured: 351 (BmsLimits), 355 (BmsSOC), 356 (BmsStatus1), 35A (BmsErrors), 35B (BmsStatus2), 125 (DU1Feedback), 126 (DU1Status)")
	log.Println("Decoded data is written to data/ev_data.json and data/main_data.json")

	// Keep the program running
	select {}
}
