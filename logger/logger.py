#!/usr/bin/env python3
"""
CAN Bus Logger for Raspberry Pi 5
Supports both CAN0 listening and SKV file streaming
"""

import logging
from logging.handlers import RotatingFileHandler
from datetime import datetime
import tomllib
from pathlib import Path
import time
import threading
import csv
from typing import Optional
import can
import re
from can_parser import OrionCANParser

class CANMessage:
    """Represents a CAN message"""
    def __init__(self, timestamp: float, can_id: str, dlc: int, data: list[int], source: str):
        self.timestamp = timestamp
        self.can_id = can_id
        self.dlc = dlc
        self.data = data
        self.source = source
    
    def __str__(self):
        data_hex = ' '.join([f"{byte:02X}" for byte in self.data])
        return f"[{self.source}] {self.timestamp:.6f} {self.can_id} [{self.dlc}] {data_hex}"

class CANLogger:
    """Enhanced CAN logger with structured parsing"""
    def __init__(self, config=None):
        self.config = config or load_config()
        self.logger = setup_logger(self.config)
        self.parser = OrionCANParser()
        self.parsed_messages = []
        self.data_directory = Path(self.config.get("data_directory", "data/"))
        self.data_directory.mkdir(exist_ok=True)
        
    def on_can_message(self, can_msg: CANMessage):
        """Enhanced callback function for CAN messages with parsing"""
        # Parse the message using the DBC parser
        parsed_msg = self.parser.parse_message(
            can_msg.can_id,
            bytes(can_msg.data),
            can_msg.timestamp,
            can_msg.dlc
        )
        
        if parsed_msg:
            self.parsed_messages.append(parsed_msg)
            self.logger.info(f"Parsed: {parsed_msg.can_id} -> {parsed_msg.signals}")
            
            # Save structured JSON periodically (every 10 messages)
            if len(self.parsed_messages) % 10 == 0:
                self.save_structured_data()
        else:
            self.logger.info(f"Unparsed: {can_msg}")
    
    def save_structured_data(self):
        """Save parsed messages to structured JSON file"""
        if not self.parsed_messages:
            return
            
        try:
            # Create structured JSON
            json_data = self.parser.create_structured_json(
                self.parsed_messages,
                mode="live"
            )
            
            # Save to file
            output_file = self.data_directory / "bms_data.json"
            if self.parser.save_json(json_data, str(output_file)):
                self.logger.info(f"Saved structured data to {output_file}")
            else:
                self.logger.error(f"Failed to save structured data to {output_file}")
                
        except Exception as e:
            self.logger.error(f"Error saving structured data: {e}")
    
    def cleanup(self):
        """Save any remaining data before shutdown"""
        self.save_structured_data()

def parse_size(size_str: str) -> int:
    """Parse size string like '10MB' to bytes"""
    size_str = size_str.upper().strip()
    if size_str.endswith('KB'):
        return int(size_str[:-2]) * 1024
    elif size_str.endswith('MB'):
        return int(size_str[:-2]) * 1024 * 1024
    elif size_str.endswith('GB'):
        return int(size_str[:-2]) * 1024 * 1024 * 1024
    else:
        return int(size_str)

def load_config():
    """Load configuration from pyproject.toml"""
    try:
        with open("pyproject.toml", "rb") as f:
            config = tomllib.load(f)
        return config.get("tool", {}).get("logger", {})
    except Exception as e:
        print(f"Warning: Could not load config: {e}")
        return {}

def setup_logger(config=None):
    """Setup basic logging configuration with rotation"""
    if config is None:
        config = load_config()
    
    log_level = getattr(logging, config.get("log_level", "INFO"))
    log_file = config.get("log_file", "can_bus.log")
    max_log_size = parse_size(config.get("max_log_size", "10MB"))
    backup_count = config.get("backup_count", 5)
    
    # Create rotating file handler
    rotating_handler = RotatingFileHandler(
        log_file,
        maxBytes=max_log_size,
        backupCount=backup_count,
        encoding='utf-8'
    )
    
    # Create console handler
    console_handler = logging.StreamHandler()
    
    # Create formatter
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Set formatter for both handlers
    rotating_handler.setFormatter(formatter)
    console_handler.setFormatter(formatter)
    
    # Create logger and add handlers
    logger = logging.getLogger('CANLogger')
    logger.setLevel(log_level)
    
    # Clear any existing handlers to avoid duplicates
    logger.handlers.clear()
    
    # Add handlers
    logger.addHandler(rotating_handler)
    logger.addHandler(console_handler)
    
    # Log rotation info
    logger.info(f"Log rotation configured: max size {config.get('max_log_size', '10MB')}, {backup_count} backups")
    
    return logger

def parse_candump_line(line: str) -> Optional[CANMessage]:
    """Parse a single candump line into a CANMessage object"""
    try:
        # candump format: (timestamp) interface can_id [dlc] data...
        # Example: (1755680927.992924)  can0  6B0   [8]  00 00 0A E6 A0 05 86 D3
        
        # Extract timestamp
        timestamp_match = re.search(r'\(([\d\.]+)\)', line)
        if not timestamp_match:
            return None
        timestamp = float(timestamp_match.group(1))
        
        # Extract interface, can_id, dlc
        parts = line.strip().split()
        if len(parts) < 4:
            return None
        
        interface = parts[1]
        can_id = parts[2]
        dlc = int(parts[3].strip('[]'))
        
        # Parse data bytes (everything after dlc)
        data = []
        for i in range(4, len(parts)):
            if parts[i].strip():
                data.append(int(parts[i], 16))
        
        return CANMessage(timestamp, can_id, dlc, data, interface)
    except Exception as e:
        print(f"Error parsing candump line: {e}")
        return None

def parse_skv_line(line: str) -> Optional[CANMessage]:
    """Parse a single SKV line into a CANMessage object"""
    try:
        parts = line.strip().split(';')
        if len(parts) < 5:
            return None
        
        timestamp = float(parts[0])
        interface = parts[1]
        can_id = parts[2]
        dlc = int(parts[3].strip('[]'))
        
        # Parse data bytes
        data = []
        for i in range(4, min(12, len(parts))):
            if parts[i].strip():
                data.append(int(parts[i], 16))
        
        return CANMessage(timestamp, can_id, dlc, data, interface)
    except Exception as e:
        print(f"Error parsing SKV line: {e}")
        return None

def stream_can_file(file_path: str, delay: float, logger, callback):
    """Stream CAN messages from file (supports both SKV and candump formats)"""
    try:
        with open(file_path, 'r') as f:
            # Try to detect format by reading first line
            first_line = f.readline().strip()
            f.seek(0)  # Reset to beginning
            
            # Check if it's SKV format (semicolon-separated)
            if ';' in first_line:
                logger.info(f"Detected SKV format, skipping header")
                next(f)  # Skip header
                parse_func = parse_skv_line
            else:
                logger.info(f"Detected candump format")
                parse_func = parse_candump_line
            
            for line_num, line in enumerate(f, 1):
                if not line.strip():
                    continue
                
                can_msg = parse_func(line)
                if can_msg:
                    logger.info(f"CAN Stream: {can_msg}")
                    if callback:
                        callback(can_msg)
                
                time.sleep(delay)
                
    except FileNotFoundError:
        logger.error(f"CAN file not found: {file_path}")
    except Exception as e:
        logger.error(f"Error streaming CAN file: {e}")

def listen_can0(interface: str, bitrate: int, logger, callback):
    """Listen to CAN0 interface using python-can"""
    logger.info(f"Starting CAN0 listener on interface: {interface} at {bitrate} bps")
    
    try:
        # Configure CAN interface
        can.rc['interface'] = 'socketcan'
        can.rc['channel'] = interface
        can.rc['bitrate'] = bitrate
        
        # Create CAN bus
        bus = can.interface.Bus()
        logger.info(f"Successfully connected to CAN bus: {interface}")
        
        # Listen for messages
        for msg in bus:
            if msg is not None:
                # Convert python-can message to our CANMessage format
                can_msg = CANMessage(
                    timestamp=msg.timestamp,
                    can_id=f"{msg.arbitration_id:X}",
                    dlc=msg.dlc,
                    data=list(msg.data),
                    source=interface
                )
                
                logger.info(f"CAN0: {can_msg}")
                if callback:
                    callback(can_msg)
                    
    except Exception as e:
        logger.error(f"Error listening to CAN0: {e}")
        logger.error("Make sure CAN interface is up: sudo ip link set can0 up type can bitrate 500000")
    finally:
        try:
            bus.shutdown()
        except:
            pass

def on_can_message(can_msg: CANMessage):
    """Callback function for CAN messages"""
    # This function will be called whenever a CAN message is received
    # You can add custom processing here
    pass

def main():
    """Main function - supports both CAN0 and SKV streaming"""
    config = load_config()
    logger = setup_logger(config)
    
    logger.info("=== CAN Bus Logger Starting ===")
    logger.info(f"Timestamp: {datetime.now()}")
    logger.info("Hello World from CAN Bus Logger!")
    
    # Display configuration
    logger.info(f"CAN Interface: {config.get('can_interface', 'can0')}")
    logger.info(f"CAN Bitrate: {config.get('can_bitrate', 500000)}")
    logger.info(f"Enable CAN0: {config.get('enable_can0', True)}")
    logger.info(f"Enable SKV Stream: {config.get('enable_skv_stream', True)}")
    logger.info(f"SKV File: {config.get('skv_file_path', '../sample/can0.skv')}")
    logger.info(f"Log File: {config.get('log_file', 'can_bus.log')}")
    logger.info(f"Max Log Size: {config.get('max_log_size', '10MB')}")
    logger.info(f"Backup Count: {config.get('backup_count', 5)}")
    
    # Create enhanced CAN logger
    can_logger = CANLogger(config)
    
    # Start SKV streaming in a separate thread
    if config.get('enable_skv_stream', True):
        skv_file = config.get('skv_file_path', '../sample/can0.skv')
        skv_delay = config.get('skv_stream_delay', 0.1)
        
        skv_thread = threading.Thread(
            target=stream_can_file,
            args=(skv_file, skv_delay, logger, can_logger.on_can_message),
            daemon=True
        )
        skv_thread.start()
        logger.info(f"Started CAN file streaming from {skv_file}")
    
    # Start CAN0 listening in a separate thread
    if config.get('enable_can0', True):
        can_interface = config.get('can_interface', 'can0')
        can_bitrate = config.get('can_bitrate', 500000)
        
        can_thread = threading.Thread(
            target=listen_can0,
            args=(can_interface, can_bitrate, logger, can_logger.on_can_message),
            daemon=True
        )
        can_thread.start()
        logger.info(f"Started CAN0 listening on {can_interface}")
    
    # Keep main thread alive
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        logger.info("Shutting down CAN Bus Logger...")
        can_logger.cleanup()

if __name__ == "__main__":
    main()
