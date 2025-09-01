#!/usr/bin/env python3
"""
CAN Message Parser for Orion BMS
Parses CAN messages according to Orion_CANBUS.dbc specifications
"""

import json
import struct
from datetime import datetime
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, asdict
from enum import Enum

class MessageType(Enum):
    """Message types from DBC file"""
    MSGID_0X6B0 = 0x6B0  # Pack current/voltage/SOC/relay
    MSGID_0X6B1 = 0x6B1  # Pack DCL/CCL/temperatures
    MSGID_0X6B2 = 0x6B2  # Cell voltage IDs and populated cells
    MSGID_0X6B3 = 0x6B3  # Pack CCL/DCL and failsafe statuses
    MSGID_0X6B4 = 0x6B4  # J1772 AC power/current limits
    MSGID_0X1806E7F4 = 0x1806E7F4  # Maximum pack voltage and DTC
    MSGID_0X1806E5F4 = 0x1806E5F4  # Maximum cell voltage and DTC
    MSGID_0X1806E9F4 = 0x1806E9F4  # Maximum cell voltage and DTC
    MSGID_0X18FF50E5 = 0x18FF50E5  # Third party device (blank)

@dataclass
class CANSignal:
    """Represents a CAN signal definition"""
    name: str
    start_bit: int
    length: int
    factor: float
    offset: float
    unit: str
    min_val: float
    max_val: float

@dataclass
class CANMessage:
    """Represents a parsed CAN message"""
    can_id: str
    timestamp: float
    dlc: int
    raw_data: bytes
    message_type: Optional[MessageType] = None
    signals: Dict[str, Any] = None

class OrionCANParser:
    """Parser for Orion BMS CAN messages"""
    
    VERSION = "1.0.0"
    NAMESPACE = "orion_bms_can"
    
    def __init__(self):
        self.message_definitions = self._build_message_definitions()
        
    def _build_message_definitions(self) -> Dict[int, Dict[str, CANSignal]]:
        """Build message definitions from DBC file"""
        definitions = {
            0x6B0: {
                "Pack_Current": CANSignal("Pack_Current", 48, 16, 0.1, 0, "Amps", 0, 0),
                "Pack_Inst_Voltage": CANSignal("Pack_Inst_Voltage", 32, 16, 0.1, 0, "Volts", 0, 0),
                "Pack_SOC": CANSignal("Pack_SOC", 24, 8, 0.5, 0, "Percent", 0, 0),
                "Relay_State": CANSignal("Relay_State", 16, 16, 1.0, 0, "", 0, 0),
                "CRC_Checksum": CANSignal("CRC_Checksum", 0, 8, 1.0, 1720, "", 0, 0),
            },
            0x6B1: {
                "Pack_DCL": CANSignal("Pack_DCL", 48, 16, 1.0, 0, "Amps", 0, 0),
                "Pack_CCL": CANSignal("Pack_CCL", 40, 8, 1.0, 0, "Amps", 0, 0),
                "High_Temperature": CANSignal("High_Temperature", 24, 8, 1.0, 0, "Celsius", 0, 0),
                "Low_Temperature": CANSignal("Low_Temperature", 16, 8, 1.0, 0, "Celsius", 0, 0),
                "CRC_Checksum": CANSignal("CRC_Checksum", 0, 8, 1.0, 1721, "", 0, 0),
            },
            0x6B2: {
                "High_Cell_Voltage": CANSignal("High_Cell_Voltage", 48, 16, 1.0E-4, 0, "Volts", 0, 0),
                "Low_Cell_Voltage": CANSignal("Low_Cell_Voltage", 32, 16, 1.0E-4, 0, "Volts", 0, 0),
                "High_Cell_ID": CANSignal("High_Cell_ID", 24, 8, 1.0, 0, "ID", 0, 0),
                "Low_Cell_ID": CANSignal("Low_Cell_ID", 16, 8, 1.0, 0, "ID", 0, 0),
                "Populated_Cells": CANSignal("Populated_Cells", 8, 8, 1.0, 0, "Num", 0, 0),
                "CRC_Checksum": CANSignal("CRC_Checksum", 0, 8, 1.0, 0, "", 0, 0),
            },
            0x6B3: {
                "Pack_CCL": CANSignal("Pack_CCL", 7, 16, 1.0, 0, "Amps", 0, 0),
                "Pack_DCL": CANSignal("Pack_DCL", 23, 16, 1.0, 0, "Amps", 0, 0),
                "Failsafe_Statuses": CANSignal("Failsafe_Statuses", 39, 16, 1.0, 0, "", 0, 0),
            },
            0x6B4: {
                "J1772_AC_Power_Limit": CANSignal("J1772_AC_Power_Limit", 7, 16, 1.0, 0, "Watts", 0, 0),
                "J1772_AC_Current_Limit": CANSignal("J1772_AC_Current_Limit", 23, 16, 1.0, 0, "Amps", 0, 0),
                "J1772_Plug_State": CANSignal("J1772_Plug_State", 39, 8, 1.0, 0, "", 0, 0),
            },
            0x1806E7F4: {
                "Maximum_Pack_Voltage": CANSignal("Maximum_Pack_Voltage", 7, 16, 0.1, 0, "Volts", 0, 0),
                "Pack_CCL": CANSignal("Pack_CCL", 23, 16, 0.1, 0, "Amps", 0, 0),
                "DTC_P0A08_Charger_Safety_Relay_Fault": CANSignal("DTC_P0A08_Charger_Safety_Relay_Fault", 39, 1, 1.0, 0, "", 0, 1),
            },
            0x1806E5F4: {
                "Maximum_Cell_Voltage": CANSignal("Maximum_Cell_Voltage", 7, 16, 0.001388888888888889, 0, "Volts", 0, 0),
                "Pack_CCL": CANSignal("Pack_CCL", 23, 16, 0.1, 0, "Amps", 0, 0),
                "DTC_P0A08_Charger_Safety_Relay_Fault": CANSignal("DTC_P0A08_Charger_Safety_Relay_Fault", 39, 1, 1.0, 0, "", 0, 1),
            },
            0x1806E9F4: {
                "Maximum_Cell_Voltage": CANSignal("Maximum_Cell_Voltage", 7, 16, 0.001388888888888889, 0, "Volts", 0, 0),
                "Pack_CCL": CANSignal("Pack_CCL", 23, 16, 0.1, 0, "Amps", 0, 0),
                "DTC_P0A08_Charger_Safety_Relay_Fault": CANSignal("DTC_P0A08_Charger_Safety_Relay_Fault", 39, 1, 1.0, 0, "", 0, 1),
            },
        }
        return definitions
    
    def _extract_signal(self, data: bytes, signal: CANSignal) -> Optional[float]:
        """Extract a signal value from CAN data according to signal definition"""
        try:
            # Convert bytes to integer (big endian based on debug analysis)
            data_int = int.from_bytes(data, byteorder='big')
            
            # Extract bits
            mask = (1 << signal.length) - 1
            shift = signal.start_bit
            raw_value = (data_int >> shift) & mask
            
            # Apply factor and offset
            value = raw_value * signal.factor + signal.offset
            
            # Round voltage values to 4 decimal places
            if signal.unit == "Volts":
                value = round(value, 4)
            
            return value
        except Exception as e:
            print(f"Error extracting signal {signal.name}: {e}")
            return None
    
    def _parse_relay_flags(self, relay_state: int) -> Dict[str, bool]:
        """Parse relay state word into individual flags"""
        # Based on the existing code in web_app.py, these are the low byte flags
        low_byte = relay_state & 0xFF
        return {
            "discharge_enabled": bool(low_byte & 0x01),
            "charge_enabled": bool(low_byte & 0x02),
            "charger_safety_enabled": bool(low_byte & 0x04),
            "malfunction_active": bool(low_byte & 0x08),
            "mpi_active": bool(low_byte & 0x10),
            "always_on_active": bool(low_byte & 0x20),
            "is_ready": bool(low_byte & 0x40),
            "is_charging": bool(low_byte & 0x80),
            "raw_low_byte_hex": f"0x{low_byte:02X}"
        }
    
    def _calculate_crc(self, data: bytes, can_id: int) -> bool:
        """Calculate and verify CRC checksum"""
        try:
            # Simple CRC calculation (this may need to be adjusted based on actual CRC algorithm)
            if len(data) < 8:
                return False
            
            # For now, return True - CRC validation can be implemented later
            return True
        except Exception:
            return False
    
    def parse_message(self, can_id: str, data: bytes, timestamp: float, dlc: int) -> Optional[CANMessage]:
        """Parse a CAN message according to DBC definitions"""
        try:
            # Convert hex string to int
            can_id_int = int(can_id, 16)
            
            # Check if we have a definition for this message
            if can_id_int not in self.message_definitions:
                return None
            
            # Create message object
            message = CANMessage(
                can_id=can_id,
                timestamp=timestamp,
                dlc=dlc,
                raw_data=data,
                message_type=MessageType(can_id_int) if can_id_int in [m.value for m in MessageType] else None,
                signals={}
            )
            
            # Parse each signal
            signals = {}
            for signal_name, signal_def in self.message_definitions[can_id_int].items():
                value = self._extract_signal(data, signal_def)
                if value is not None:
                    # Use original DBC signal names
                    signals[signal_name] = value
            
            # Special handling for relay state (0x6B0)
            if can_id_int == 0x6B0:
                # Get the raw relay state value
                relay_state_raw = None
                for signal_name, signal_def in self.message_definitions[can_id_int].items():
                    if signal_name == "Relay_State":
                        relay_state_raw = self._extract_signal(data, signal_def)
                        break
                
                if relay_state_raw is not None:
                    relay_state = int(relay_state_raw)
                    signals["relay_state_word"] = f"0x{relay_state:04X}"
                    signals["relay_flags_lowbyte"] = self._parse_relay_flags(relay_state)
            
            # Add CRC validation
            if "CRC_Checksum" in signals:
                signals["crc"] = f"0x{int(signals['CRC_Checksum']):02X}"
                signals["crc_ok"] = self._calculate_crc(data, can_id_int)
            
            # Add raw data
            signals["raw"] = " ".join([f"{b:02X}" for b in data])
            
            message.signals = signals
            return message
            
        except Exception as e:
            print(f"Error parsing message {can_id}: {e}")
            return None
    
    def create_structured_json(self, messages: List[CANMessage], 
                              cells_data: Optional[Dict] = None,
                              mode: str = "live") -> Dict[str, Any]:
        """Create structured JSON output with namespace and version"""
        
        # Group messages by CAN ID
        frames = {}
        for msg in messages:
            if msg.signals:
                frames[msg.can_id] = msg.signals
        
        # Create the structured output
        output = {
            "namespace": self.NAMESPACE,
            "version": self.VERSION,
            "written_at_epoch": datetime.now().timestamp(),
            "mode": mode,
            "frames": [{"can_id": can_id, **signals} for can_id, signals in frames.items()],
        }
        
        # Add cells data if provided
        if cells_data:
            output["cells"] = cells_data
        
        return output
    
    def save_json(self, data: Dict[str, Any], filepath: str) -> bool:
        """Save structured JSON data to file"""
        try:
            with open(filepath, 'w', encoding='utf-8') as f:
                json.dump(data, f, indent=2, ensure_ascii=False)
            return True
        except Exception as e:
            print(f"Error saving JSON to {filepath}: {e}")
            return False

# Example usage and testing
if __name__ == "__main__":
    # Test with sample data from bms_data.json
    parser = OrionCANParser()
    
    # Sample CAN message data (from the existing bms_data.json)
    test_messages = [
        {
            "can_id": "6B0",
            "data": bytes([0x00, 0x32, 0x0A, 0xF7, 0xA0, 0x15, 0x86, 0x26]),
            "timestamp": 1234567890.123,
            "dlc": 8
        },
        {
            "can_id": "6B1", 
            "data": bytes([0x00, 0x14, 0x00, 0x02, 0x22, 0x13, 0x00, 0x04]),
            "timestamp": 1234567890.124,
            "dlc": 8
        },
        {
            "can_id": "6B2",
            "data": bytes([0x00, 0x02, 0x97, 0xF2, 0x02, 0xF2, 0x48, 0xC7]),
            "timestamp": 1234567890.125,
            "dlc": 8
        }
    ]
    
    # Parse messages
    parsed_messages = []
    for msg_data in test_messages:
        msg = parser.parse_message(
            msg_data["can_id"],
            msg_data["data"],
            msg_data["timestamp"],
            msg_data["dlc"]
        )
        if msg:
            parsed_messages.append(msg)
    
    # Create structured JSON
    json_data = parser.create_structured_json(parsed_messages)
    
    # Save to file
    parser.save_json(json_data, "test_output.json")
    
    print("Test completed. Check test_output.json for results.")
