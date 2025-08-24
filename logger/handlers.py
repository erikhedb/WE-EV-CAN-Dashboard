# Handlers for specific CAN IDs (0x6B0, 0x6B1, 0x6B2) and default
import time

def handle_6b0(data, can_id):
    return {
        "can_id": f"0x{can_id:X}",
        "current_A": round(int.from_bytes(data[0:2], 'big') * 0.1, 4),
        "voltage_V": round(int.from_bytes(data[2:4], 'big') * 0.1, 4),
        "soc_percent": round(data[4] * 0.5, 2),
        "relay_state": f"0x{data[5]<<8|data[6]:04X}",
        "crc": data[7],
        "timestamp": time.time()
    }

def handle_6b1(data, can_id):
    return {
        "can_id": f"0x{can_id:X}",
        "dcl_A": int.from_bytes(data[0:2], 'big'),
        "ccl_A": data[2],
        "high_temp_C": data[4],
        "low_temp_C": data[5],
        "crc": data[7],
        "timestamp": time.time()
    }

def handle_6b2(data, can_id):
    return {
        "can_id": f"0x{can_id:X}",
        "high_cell_voltage_V": round(int.from_bytes(data[0:2], 'big') * 1e-4, 4),
        "low_cell_voltage_V": round(int.from_bytes(data[2:4], 'big') * 1e-4, 4),
        "high_cell_id": data[4],
        "low_cell_id": data[5],
        "populated_cells": data[6],
        "crc": data[7],
        "timestamp": time.time()
    }

def handle_default(data, can_id):
    return {
        "can_id": f"0x{can_id:X}",
        "data": data.hex(),
        "crc": data[7],
        "timestamp": time.time()
    }

handlers = {
    0x6B0: handle_6b0,
    0x6B1: handle_6b1,
    0x6B2: handle_6b2,
    'default': handle_default,
}
