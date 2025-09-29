# WE-EV-CAN-Dashboard

This project is a modular CAN bus dashboard system for a DIY EV build using a Raspberry Pi and Orion BMS 2. It is written in Go and uses NATS as the internal message bus to decouple components.

![Sample UI](sample-ui.png)



## 🧩 Architecture Overview

The project is split into three main services:

1. **CAN Reader**

   * Listens on `can0`
   * Publishes raw CAN frames (with timestamps) to NATS (`can.raw`)
   * Logs all raw traffic to JSON-formatted logfile

2. **Message Handler**

   * Subscribes to `can.raw`
   * Decodes / processes messages
   * Saves parsed messages to JSON for frontend use

3. **Web UI**

   * Reads JSON files
   * Presents data via a basic HTML/JS frontend

4. **Relpay**

```
go build -o bin/replay ./cmd/replay
./bin/replay
```


   * To replay the messages on the logs/can_raw_sample.log file and put them on the message buss

---

## 🚀 Getting Started

### 📦 Prerequisites

* Go 1.21+
* Docker (for running NATS)
* SocketCAN support (Linux)
* CAN interface set up as `can0`

### 🔧 Installation

Clone the repo:

```bash
git clone https://github.com/YOUR_USERNAME/WE-EV-CAN-Dashboard.git
cd WE-EV-CAN-Dashboard
```

Install Go dependencies:

```bash
go mod tidy
```

---

## 🛰️ Running NATS (Local Dev)

### Option 1: Docker (Recommended)

```bash
docker run -p 4222:4222 --name nats-server -ti nats:latest
```

### Option 2: Native Install (macOS/Linux)

```bash
brew install nats-server
nats-server
```

### Option 3: Docker-compose
```bash
> docker compose up -d
> docker compose down
> docker compose logs -f
```



---

## ⚙️ Compile and Run Services

### 1. CAN Reader

Build and run:

```bash
cd cmd/reader
go build -o can-reader
./can-reader
```

### 2. Message Handler *(coming soon)*

### 3. UI *(coming soon)*

---

## 🧪 Testing

You can test the system using replay files (to be implemented) or simulated CAN traffic.

---

## 📁 Config File

Create a `config.yaml` in the project root:

```yaml
logs:
  raw_file: logs/can_raw_20250916.log
  service_log: logs/reader_service.log
```

---

## 📜 License

MIT License

# Install

## CAN Bus System Services Setup

### File Locations
- **Reader executable**: `/home/erwa/Projects/WE-EV-CAN-Dashboard/canbus-reader`
- **Handler executable**: `/home/erwa/Projects/WE-EV-CAN-Dashboard/canbus-handler`
- **UI executable**: `/home/erwa/Projects/WE-EV-CAN-Dashboard/canbus-ui`
- **Log file**: `/home/erwa/Projects/WE-EV-CAN-Dashboard/logs/canbus.json` (when `-l` flag is used)
- **Reader service**: `/etc/systemd/system/canbus-reader.service`
- **Handler service**: `/etc/systemd/system/canbus-handler.service`
- **UI service**: `/etc/systemd/system/canbus-ui.service`
- **Configuration**: `/home/erwa/Projects/WE-EV-CAN-Dashboard/config.yaml`

### Build and Install Services
```bash
# Build all binaries
go build -o canbus-reader ./cmd/reader
go build -o canbus-handler ./cmd/handler
go build -o canbus-ui ./cmd/ui
chmod +x canbus-reader canbus-handler canbus-ui

# Install systemd services
sudo cp canbus-reader.service /etc/systemd/system/
sudo cp canbus-handler.service /etc/systemd/system/
sudo cp canbus-ui.service /etc/systemd/system/
sudo systemctl daemon-reload

# Enable and start all services
sudo systemctl enable canbus-reader canbus-handler canbus-ui
sudo systemctl start canbus-reader canbus-handler canbus-ui
```

### Service Management
```bash
# Check all service status
sudo systemctl status canbus-reader canbus-handler canbus-ui

# View live logs
sudo journalctl -u canbus-reader -f
sudo journalctl -u canbus-handler -f
sudo journalctl -u canbus-ui -f

# Stop all services
sudo systemctl stop canbus-reader canbus-handler canbus-ui

# Start all services
sudo systemctl start canbus-reader canbus-handler canbus-ui

# Restart all services
sudo systemctl restart canbus-reader canbus-handler canbus-ui

# Disable auto-start on boot
sudo systemctl disable canbus-reader canbus-handler canbus-ui
```

### Access the Web UI
Once all services are running, access the dashboard at: http://localhost:8080