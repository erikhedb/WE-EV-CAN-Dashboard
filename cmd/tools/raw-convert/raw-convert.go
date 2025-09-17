// ***************************************************************************
// Converts a raw CAN log file to a structured JSON format.
// Usage: raw-convert <raw_input_file> <output_file>
// The output file will be overwritten if it already exists.
//
// # By Erik Wastlin in 2025
//
// Use this to create JSON files from raw CAN logs for easier testing and analysis.
//
// ***************************************************************************
package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strconv"
)

type CANMessage struct {
	ID     string `json:"id"`
	Length int    `json:"length"`
	Data   string `json:"data"`
	Meta   string `json:"meta"`
}

func main() {

	// Expecting two arguments: input raw file and output JSON file
	if len(os.Args) < 3 {
		log.Fatal("Usage: raw-convert <raw_input_file> <output_file>")
	}
	inputFileName := os.Args[1]
	outputFileName := os.Args[2]

	// Open input file
	inputFile, err := os.Open(inputFileName)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer inputFile.Close()

	// Create output file and overwrite if exists
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	scanner := bufio.NewScanner(inputFile)

	// Process each line in the input file
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) < 8 {
			log.Printf("Skipping line (too short): %s", line)
			continue
		}

		id := line[1:4]        // 3-byte ID (6 hex chars)
		lengthHex := line[4:5] // 1-byte length (2 hex chars)
		length, err := strconv.ParseUint(lengthHex, 16, 8)

		if err != nil || length == 0 {
			log.Printf("Skipping line (invalid length): %s", line)
			continue
		}

		data := line[5 : 5+length*2] // Data bytes (length * 2 hex chars)
		meta := line[5+length*2:]    // Remaining metadata

		msg := CANMessage{
			ID:     id,
			Length: int(length),
			Data:   data,
			Meta:   meta,
		}

		jsonBytes, _ := json.Marshal(msg)
		writer.WriteString(string(jsonBytes) + "\n")
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
}
