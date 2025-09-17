package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <raw_file>\n", os.Args[0])
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
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "t") || len(line) < 4 {
			continue
		}
		idHex := line[1:4]
		counts[strings.ToUpper(idHex)]++
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
		os.Exit(1)
	}

	// Sort and print by ID
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		iVal, _ := strconv.ParseInt(ids[i], 16, 32)
		jVal, _ := strconv.ParseInt(ids[j], 16, 32)
		return iVal < jVal
	})

	fmt.Printf("%-6s %s\n", "ID", "Count")
	fmt.Println("----------------")
	for _, id := range ids {
		fmt.Printf("%-6s %d\n", id, counts[id])
	}
}
