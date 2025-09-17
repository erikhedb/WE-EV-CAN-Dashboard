package main

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Drain()

	subject := "can.raw"
	_, err = nc.Subscribe(subject, func(msg *nats.Msg) {
		fmt.Println(string(msg.Data))
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to subject '%s': %v", subject, err)
	}

	log.Printf("Listening on subject '%s'. Press Ctrl+C to exit.\n", subject)
	select {}
}
