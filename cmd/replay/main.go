//***************************************************************************
// Replays a JSONL CAN log file onto the NATS bus (subject: can.raw) using delta timing from the meta field.
// Usage: replay [-c] <log_file.jsonl>
// If the -c flag is provided, it will continuously loop the file.
//
// The meta field in each JSON message contains the delta time in milliseconds since the last message.
// This allows for accurate replay of CAN bus timing patterns.
//
// By Erik Wästlin in 2025
//
// Use this to simulate CAN traffic for testing and development.
//***************************************************************************

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// CANMessage represents a CAN message from the log file
type CANMessage struct {
	ID     string `json:"id"`
	Length int    `json:"length"`
	Data   string `json:"data"`
	Meta   int    `json:"meta"` // Delta time in milliseconds
}

func main() {
	// Parse arguments
	continuous := false
	logFilePath := ""
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "-c" {
			continuous = true
		} else if logFilePath == "" {
			logFilePath = arg
		}
	}
	if logFilePath == "" {
		fmt.Printf("Usage: %s [-c] <log_file.jsonl>\n", os.Args[0])
		os.Exit(1)
	}

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Drain()

	log.Printf("Starting CAN replay from file: %s", logFilePath)
	if continuous {
		log.Println("Continuous mode enabled - will loop indefinitely")
	}

	replayCount := 0
	for {
		replayCount++
		if continuous {
			log.Printf("Starting replay iteration %d", replayCount)
		}

		file, err := os.Open(logFilePath)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}

		scanner := bufio.NewScanner(file)
		messageCount := 0

		for scanner.Scan() {
			line := scanner.Text()
			messageCount++

			// Parse the JSON to extract the meta (delta time) field
			var canMsg CANMessage
			err := json.Unmarshal([]byte(line), &canMsg)
			if err != nil {
				log.Printf("Failed to parse JSON message %d: %v", messageCount, err)
				log.Printf("Raw line: %s", line)
				continue
			}

			// Wait for the delta time specified in the meta field (in milliseconds)
			if canMsg.Meta > 0 {
				sleepDuration := time.Duration(canMsg.Meta) * time.Millisecond
				if canMsg.Meta > 100 { // Only log longer delays to reduce spam
					log.Printf("Message %d (ID: %s): Waiting %dms", messageCount, canMsg.ID, canMsg.Meta)
				}
				time.Sleep(sleepDuration)
			}

			// Publish the original JSON message to the bus
			err = nc.Publish("can.raw", []byte(line))
			if err != nil {
				log.Printf("Failed to publish message %d: %v", messageCount, err)
				continue
			}

			// Log every 50th message to show progress without spam
			if messageCount%50 == 0 {
				log.Printf("Published %d messages (latest ID: %s)", messageCount, canMsg.ID)
			}
		}

		file.Close()

		if err := scanner.Err(); err != nil {
			log.Fatalf("Scanner error: %v", err)
		}

		log.Printf("Completed replay iteration %d: %d messages sent", replayCount, messageCount)

		if !continuous {
			break
		}

		// Small pause between iterations in continuous mode
		log.Println("Pausing 1 second before next iteration...")
		time.Sleep(1 * time.Second)
	}

	log.Println("Replay complete.")
}
