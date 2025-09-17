//***************************************************************************
// Replays a JSONL CAN log file onto the NATS bus (subject: can.raw) with a 10ms delay between messages.
// Usage: replay [-c] <log_file.jsonl>
// If the -c flag is provided, it will continuously loop the file.
//
// By Erik Wästlin in 2025
//
// Use this to simulate CAN traffic for testing and development.
//***************************************************************************

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

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

	for {
		file, err := os.Open(logFilePath)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()
			// Just put the JSON input on the bus as-is
			err = nc.Publish("can.raw", []byte(line))
			if err != nil {
				log.Printf("Failed to publish message: %v", err)
				continue
			}
			time.Sleep(10 * time.Millisecond)
		}

		file.Close()

		if err := scanner.Err(); err != nil {
			log.Fatalf("Scanner error: %v", err)
		}

		if !continuous {
			break
		}
	}

	log.Println("Replay complete.")
}
