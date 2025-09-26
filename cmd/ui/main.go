package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Paths struct {
		DataFolder     string `yaml:"data_folder"`
		LogFolder      string `yaml:"log_folder"`
		UIStaticFolder string `yaml:"ui_static_folder"`
	} `yaml:"paths"`
	Server struct {
		UIPort int `yaml:"ui_port"`
	} `yaml:"server"`
}

func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build paths
	dataPath := filepath.Join(config.Paths.DataFolder, "cell_data.json")
	staticPath := config.Paths.UIStaticFolder

	log.Printf("Config loaded from: %s", configPath)
	log.Printf("Data path: %s", dataPath)
	log.Printf("Static path: %s", staticPath)
	log.Printf("UI port: %d", config.Server.UIPort)

	// Serve cell_data.json as the main API endpoint
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(dataPath)
		if err != nil {
			log.Printf("Error opening cell_data.json: %v", err)
			http.Error(w, "cell_data.json not available", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, f) // stream file content as-is
	})

	// Serve static files (index.html etc.)
	fs := http.FileServer(http.Dir(staticPath))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", config.Server.UIPort)
	log.Printf("UI server running on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
