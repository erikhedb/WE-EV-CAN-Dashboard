package main

// This program listens on the CAN interface (can0), publishes all CAN frames to a NATS subject (can.raw),
// and optionally logs them in JSON format to a file specified via -l flag. The service log only records start and stop times.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.einride.tech/can/pkg/socketcan"
)

type CANFrame struct {
	ID     string `json:"id"`
	Length int    `json:"length"`
	Data   string `json:"data"`
	Meta   int64  `json:"meta"`
}

func main() {
	// Command-line flag for logging
	logFileFlag := flag.String("l", "", "Optional log file for writing JSON messages")
	flag.Parse()

	// Load configuration
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	serviceLogPath := viper.GetString("logs.service_log")

	// Setup service logger (append mode, keep between runs)
	serviceLogFile, err := os.OpenFile(serviceLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open service log file: %v", err)
	}
	defer serviceLogFile.Close()
	log.SetOutput(serviceLogFile)

	// Record start time
	log.Printf("Reader service started at %s", time.Now().Format(time.RFC3339))

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	// Setup JetStream with max age of 60 seconds
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Error initializing JetStream: %v", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "CAN",
		Subjects: []string{"can.raw"},
		MaxAge:   60 * time.Second,
	})
	if err != nil {
		log.Fatalf("Error creating JetStream stream: %v", err)
	}

	// Open CAN interface
	conn, err := socketcan.DialContext(context.Background(), "can", "can0")
	if err != nil {
		log.Fatalf("Failed to open CAN interface: %v", err)
	}
	defer conn.Close()
	receiver := socketcan.NewReceiver(conn)

	// Optional JSON log file from -l flag
	var logEncoder *json.Encoder
	if *logFileFlag != "" {
		lf, err := os.Create(*logFileFlag)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		defer lf.Close()
		logEncoder = json.NewEncoder(lf)
	}

	// Mark program start time
	startTime := time.Now()

	for receiver.Receive() {
		frame := receiver.Frame()

		wrapped := CANFrame{
			ID:     fmt.Sprintf("%X", frame.ID),
			Length: len(frame.Data[:frame.Length]),
			Data:   fmt.Sprintf("%X", frame.Data[:frame.Length]),
			Meta:   time.Since(startTime).Milliseconds(),
		}

		encoded, err := json.Marshal(wrapped)
		if err != nil {
			continue
		}

		// Publish to NATS JetStream
		_, _ = js.Publish("can.raw", encoded)

		// Write to JSON log file if enabled
		if logEncoder != nil {
			_ = logEncoder.Encode(wrapped)
		}
	}

	// On exit, record stop time
	if err := receiver.Err(); err != nil {
		log.Printf("Reader service stopped with error at %s: %v", time.Now().Format(time.RFC3339), err)
	} else {
		log.Printf("Reader service stopped cleanly at %s", time.Now().Format(time.RFC3339))
	}
}
