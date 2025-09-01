# CAN Bus Logger

A Python-based CAN bus logger for Raspberry Pi 5 with support for both real CAN bus listening and SKV file streaming.

## Features

- **Dual Data Sources**: CAN0 interface listening and SKV file streaming
- **Log Rotation**: Automatic log file rotation (10MB max, 5 backups)
- **Cross-Platform**: Works on Raspberry Pi (CAN0) and Mac (SKV)
- **Configuration-Driven**: All settings in `pyproject.toml`
- **Multi-threaded**: Both data sources run independently

## Configuration

Edit `pyproject.toml` to switch between modes:

### Mac Development Mode
```toml
enable_can0 = false
enable_skv_stream = true
```

### Raspberry Pi Production Mode
```toml
enable_can0 = true
enable_skv_stream = false
```

## Usage

```bash
# Install dependencies
uv sync

# Run logger
python logger.py
```

## Files

- `logger.py` - Main logger application
- `pyproject.toml` - Configuration and dependencies
- `sample/can0.skv` - Sample CAN bus data file
- `can_bus.log` - Generated log file (with rotation)

## Dependencies

- `python-can>=4.0.0` - For CAN bus interface (Raspberry Pi only)
- Built-in Python modules: `logging`, `threading`, `tomllib`
