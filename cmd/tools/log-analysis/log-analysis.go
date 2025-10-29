package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type canMessage struct {
	ID string `json:"id"`
}

var idDescriptions = map[string]string{
	"036": "Battery Cell Broadcast",
	"076": "Thermistor Broadcast",
	"125": "DU1Feedback (drive unit telemetry)",
	"126": "DU1Status (drive unit state)",
	"127": "DU1Diagnostic (limit flags)",
	"351": "BmsLimits (charge/discharge ceilings)",
	"355": "BmsSOC (state-of-charge metrics)",
	"356": "BmsStatus1 (pack V/I/temp)",
	"357": "BMSCCSCommands (charging commands)",
	"35A": "BmsErrors (fault bitfield)",
	"35B": "BmsStatus2 (relay states + isolation)",
	"6B0": "Battery Pack Status",
	"6B1": "High Cell Information",
	"6B2": "Low Cell Information",
	"6B3": "Temperature Information",
	"6B4": "System Control Information",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <canbus_json>\n", os.Args[0])
		os.Exit(1)
	}

	inputPath := os.Args[1]
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	counts := make(map[string]int)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg canMessage
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &msg); err != nil {
			fmt.Fprintf(os.Stderr, "Skipping invalid JSON: %v\n", err)
			continue
		}
		if msg.ID == "" {
			continue
		}
		counts[strings.ToUpper(msg.ID)]++
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
		os.Exit(1)
	}

	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		iVal, _ := strconv.ParseInt(ids[i], 16, 32)
		jVal, _ := strconv.ParseInt(ids[j], 16, 32)
		return iVal < jVal
	})

	fmt.Printf("%-6s %-7s %s\n", "ID", "Count", "Description")
	fmt.Println("----------------------------------------------")
	for _, id := range ids {
		desc := idDescriptions[id]
		if desc == "" {
			desc = "-"
		}
		fmt.Printf("%-6s %-7d %s\n", id, counts[id], desc)
	}
}
