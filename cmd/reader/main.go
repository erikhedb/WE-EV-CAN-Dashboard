package reader

func main() {
	// The main logic is commented out to allow the project to build without go-socketcan.
	// Uncomment and restore when go-socketcan is available and working.
	//
	// // Load configuration
	// viper.SetConfigName("config")
	// viper.SetConfigType("yaml")
	// viper.AddConfigPath(".")
	//
	// if err := viper.ReadInConfig(); err != nil {
	//     log.Fatalf("Error reading config file: %v", err)
	// }
	//
	// rawLogPath := viper.GetString("logs.raw_file")
	// serviceLogPath := viper.GetString("logs.service_log")
	//
	// // Setup service logger
	// serviceLogFile, err := os.OpenFile(serviceLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	// if err != nil {
	//     log.Fatalf("Failed to open service log file: %v", err)
	// }
	// defer serviceLogFile.Close()
	// log.SetOutput(serviceLogFile)
	//
	// nc, err := nats.Connect(nats.DefaultURL)
	// if err != nil {
	//     log.Fatalf("Error connecting to NATS: %v", err)
	// }
	// defer nc.Close()
	//
	// rawConn, err := socketcan.NewInterface("can0")
	// if err != nil {
	//     log.Fatalf("Failed to open CAN interface: %v", err)
	// }
	//
	// log.Println("Listening on can0 and publishing to NATS (subject: can.raw)...")
	//
	// // Prepare raw log file
	// if err := os.MkdirAll(filepath.Dir(rawLogPath), 0755); err != nil {
	//     log.Fatalf("Failed to create log directory: %v", err)
	// }
	// rawLogFile, err := os.Create(rawLogPath)
	// if err != nil {
	//     log.Fatalf("Failed to create raw log file: %v", err)
	// }
	// defer rawLogFile.Close()
	//
	// rawEncoder := json.NewEncoder(rawLogFile)
	//
	// for {
	//     frame, err := rawConn.Read()
	//     if err != nil {
	//         log.Printf("Error reading CAN frame: %v", err)
	//         continue
	//     }
	//
	//     wrapped := CANFrame{
	//         Timestamp: time.Now(),
	//         ID:        frame.ID,
	//         Data:      frame.Data,
	//     }
	//
	//     encoded, err := json.Marshal(wrapped)
	//     if err != nil {
	//         log.Printf("Error encoding CAN frame: %v", err)
	//         continue
	//     }
	//
	//     // Publish to NATS
	//     err = nc.Publish("can.raw", encoded)
	//     if err != nil {
	//         log.Printf("Error publishing to NATS: %v", err)
	//     } else {
	//         log.Printf("Published CAN frame ID 0x%X to NATS", frame.ID)
	//     }
	//
	//     // Write to raw log file
	//     err = rawEncoder.Encode(wrapped)
	//     if err != nil {
	//         log.Printf("Error writing to raw log file: %v", err)
	//     }
	// }
}
