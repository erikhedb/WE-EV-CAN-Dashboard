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
	// Command-line flag for canbus logging
	enableLogging := flag.Bool("l", false, "Enable logging of CAN messages to canbus.json file")
	flag.Parse()

	// Load configuration
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	serviceLogPath := viper.GetString("logs.service_log")
	canbusJSONPath := viper.GetString("logs.canbus_json")

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

	// Setup canbus JSON file (only if logging enabled)
	var canbusEncoder *json.Encoder
	if *enableLogging {
		canbusJSONFile, err := os.Create(canbusJSONPath)
		if err != nil {
			log.Fatalf("Failed to create canbus JSON file: %v", err)
		}
		defer canbusJSONFile.Close()
		canbusEncoder = json.NewEncoder(canbusJSONFile)
	}

       // Track time of last frame
       var lastFrameTime time.Time
       for receiver.Receive() {
	       frame := receiver.Frame()

	       now := time.Now()
	       var deltaMs int64
	       if lastFrameTime.IsZero() {
		       deltaMs = 0 // First frame
	       } else {
		       deltaMs = now.Sub(lastFrameTime).Milliseconds()
	       }
	       lastFrameTime = now

	       wrapped := CANFrame{
		       ID:     fmt.Sprintf("%X", frame.ID),
		       Length: len(frame.Data[:frame.Length]),
		       Data:   fmt.Sprintf("%X", frame.Data[:frame.Length]),
		       Meta:   deltaMs,
	       }

	       encoded, err := json.Marshal(wrapped)
	       if err != nil {
		       continue
	       }

	       // Publish to NATS JetStream
	       _, _ = js.Publish("can.raw", encoded)

	       // Write to canbus JSON file if logging enabled
	       if canbusEncoder != nil {
		       _ = canbusEncoder.Encode(wrapped)
	       }
       }

	// On exit, record stop time
	if err := receiver.Err(); err != nil {
		log.Printf("Reader service stopped with error at %s: %v", time.Now().Format(time.RFC3339), err)
	} else {
		log.Printf("Reader service stopped cleanly at %s", time.Now().Format(time.RFC3339))
	}
}
