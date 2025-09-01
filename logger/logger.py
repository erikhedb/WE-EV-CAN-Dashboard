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

def stream_skv_file(skv_file_path: str, delay: float, logger, callback):
    """Stream CAN messages from SKV file"""
    try:
        with open(skv_file_path, 'r') as f:
            # Skip header
            next(f)
            
            for line_num, line in enumerate(f, 1):
                if not line.strip():
                    continue
                
                can_msg = parse_skv_line(line)
                if can_msg:
                    logger.info(f"SKV Stream: {can_msg}")
                    if callback:
                        callback(can_msg)
                
                time.sleep(delay)
                
    except FileNotFoundError:
        logger.error(f"SKV file not found: {skv_file_path}")
    except Exception as e:
        logger.error(f"Error streaming SKV file: {e}")

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
    
    # Start SKV streaming in a separate thread
    if config.get('enable_skv_stream', True):
        skv_file = config.get('skv_file_path', '../sample/can0.skv')
        skv_delay = config.get('skv_stream_delay', 0.1)
        
        skv_thread = threading.Thread(
            target=stream_skv_file,
            args=(skv_file, skv_delay, logger, on_can_message),
            daemon=True
        )
        skv_thread.start()
        logger.info(f"Started SKV streaming from {skv_file}")
    
    # Start CAN0 listening in a separate thread
    if config.get('enable_can0', True):
        can_interface = config.get('can_interface', 'can0')
        can_bitrate = config.get('can_bitrate', 500000)
        
        can_thread = threading.Thread(
            target=listen_can0,
            args=(can_interface, can_bitrate, logger, on_can_message),
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

if __name__ == "__main__":
    main()
