#!/usr/bin/env python3
import csv
import time
import struct
from datetime import datetime

import can
import isotp  # from can-isotp

# ========= CONFIG =========
CHANNEL = "can0"
BITRATE = 500000  # informational only (SocketCAN is already configured outside python)
REQUEST_ID = 0x7E3  # physical request to Orion BMS2 (your network shows 0x7E3 -> 0x7EB)
RESPONSE_ID = 0x7EB
POLL_PERIOD_S = 1.0          # how often to poll the set below
VIN_ON_START = True           # grab VIN once at startup
CSV_FILE = "orion_obd_log.csv"
TIMEOUT_S = 1.0               # per request timeout

# ========= PID/DID TABLE =========
# Fill these with your Orion BMS 2 table.
# You can mix:
# - OBD-II Mode 09 (0x09) PIDs (e.g., VIN = 0x0B on some devices; others use 0x02)
# - UDS ReadDataByIdentifier (Service 0x22) with 2-byte DID values (vendor specific).
#
# Each entry: {
#   'type': 'mode09' | 'uds22',
#   'pid' or 'did': int,
#   'name': str,
#   'parse': function(bytes)->value,
# }
#
# EXAMPLES (placeholders — replace DIDs and scales using your Orion sheet):
def u16_be(b): return struct.unpack(">H", b)[0]
def s16_be(b): return struct.unpack(">h", b)[0]
def u32_be(b): return struct.unpack(">I", b)[0]
def s32_be(b): return struct.unpack(">i", b)[0]

PID_TABLE = [
    # --- VIN via Mode 09 (choose the PID your Orion exposes; seen 0x0B in your capture) ---
    {'type': 'mode09', 'pid': 0x0B, 'name': 'vin', 'parse': lambda raw: raw.decode('ascii', errors='ignore').strip('\x00')},

    # --- Orion custom DIDs via UDS 0x22 (examples, replace DIDs + scaling with your table) ---
    # Pack instantaneous voltage (V) — EXAMPLE ONLY
    {'type': 'uds22', 'did': 0xF900, 'name': 'pack_voltage_V', 'parse': lambda raw: u16_be(raw) / 10.0},
    # Pack current (A) — EXAMPLE ONLY (signed)
    {'type': 'uds22', 'did': 0xF901, 'name': 'pack_current_A', 'parse': lambda raw: s16_be(raw) / 10.0},
    # State of charge (%) — EXAMPLE ONLY
    {'type': 'uds22', 'did': 0xF902, 'name': 'soc_percent', 'parse': lambda raw: u16_be(raw) / 10.0},
    # Relay state word — EXAMPLE ONLY (bitfield)
    {'type': 'uds22', 'did': 0xF903, 'name': 'relay_state_word', 'parse': lambda raw: f"0x{u16_be(raw):04X}"},
    # Temperature (°C) — EXAMPLE ONLY
    {'type': 'uds22', 'did': 0xF904, 'name': 'avg_temp_C', 'parse': lambda raw: s16_be(raw) / 10.0},
]

# ========= TRANSPORT SETUP =========
def make_bus_and_stack():
    bus = can.interface.Bus(channel=CHANNEL, bustype="socketcan")
    addr = isotp.Address(isotp.AddressingMode.Normal_11bits,
                         rxid=RESPONSE_ID, txid=REQUEST_ID)
    stack = isotp.CanStack(bus=bus, address=addr,
                           params={
                               'stmin': 0,            # minimum separation time
                               'blocksize': 8,        # how many CFs before FC (Orion should be fine)
                               'wftmax': 0,           # consecutive wait frames allowed
                               'tx_data_length': 8,   # classical CAN, 8 bytes
                               'tx_padding': 0x00,    # pad with zeros
                               'rx_flowcontrol_timeout': int(TIMEOUT_S * 1000),
                               'rx_consecutive_frame_timeout': int(TIMEOUT_S * 1000),
                           })
    return bus, stack

# ========= REQUEST BUILDERS =========
def build_mode09_request(pid: int) -> bytes:
    # Service 0x09, PID = 1 byte
    return bytes([0x09, pid])

def build_uds22_request(did: int) -> bytes:
    # Service 0x22, DID = 2 bytes big-endian
    return bytes([0x22, (did >> 8) & 0xFF, did & 0xFF])

# ========= SEND/RECV HELPERS (ISO-TP) =========
def xfer(stack: isotp.CanStack, payload: bytes, timeout_s: float) -> bytes:
    # Clear previous buffers
    while stack.available():
        _ = stack.recv()

    stack.send(payload)
    t0 = time.time()
    while True:
        stack.process()
        time.sleep(0.001)  # yield to kernel
        if stack.available():
            return stack.recv()
        if time.time() - t0 > timeout_s:
            raise TimeoutError("ISO-TP receive timeout")

# ========= PARSERS =========
def parse_mode09_response(pid: int, resp: bytes) -> bytes:
    # Positive response: 0x49 <pid> <data…>
    if len(resp) < 2 or resp[0] != 0x49 or resp[1] != (pid & 0xFF):
        raise ValueError(f"Unexpected Mode 09 response: {resp.hex(' ')}")
    return resp[2:]  # data after <service, pid>

def parse_uds22_response(did: int, resp: bytes) -> bytes:
    # Positive response: 0x62 <DID_H> <DID_L> <data…>
    if len(resp) < 3 or resp[0] != 0x62 or resp[1] != ((did >> 8) & 0xFF) or resp[2] != (did & 0xFF):
        raise ValueError(f"Unexpected 0x22 response: {resp.hex(' ')}")
    return resp[3:]

# ========= MAIN LOGIC =========
def ensure_csv_header(path: str, fieldnames):
    try:
        with open(path, "x", newline="") as f:
            w = csv.DictWriter(f, fieldnames=["timestamp"] + fieldnames)
            w.writeheader()
    except FileExistsError:
        pass

def main():
    bus, stack = make_bus_and_stack()

    # Build list of column names from PID_TABLE order
    columns = [entry['name'] for entry in PID_TABLE]
    ensure_csv_header(CSV_FILE, columns)

    # Optionally pull VIN once
    if VIN_ON_START:
        try:
            vin_entry = next((e for e in PID_TABLE if e['name'] == 'vin' and e['type'] == 'mode09'), None)
            if vin_entry:
                req = build_mode09_request(vin_entry['pid'])
                raw = xfer(stack, req, TIMEOUT_S)
                data = parse_mode09_response(vin_entry['pid'], raw)
                vin = vin_entry['parse'](data)
                print(f"[startup] VIN: {vin}")
        except Exception as e:
            print(f"[startup] VIN read failed: {e}")

    print("Starting poll loop. Press Ctrl+C to stop.")
    while True:
        row = {"timestamp": datetime.utcnow().isoformat() + "Z"}
        for entry in PID_TABLE:
            try:
                if entry['type'] == 'mode09':
                    req = build_mode09_request(entry['pid'])
                    raw = xfer(stack, req, TIMEOUT_S)
                    payload = parse_mode09_response(entry['pid'], raw)

                elif entry['type'] == 'uds22':
                    req = build_uds22_request(entry['did'])
                    raw = xfer(stack, req, TIMEOUT_S)
                    payload = parse_uds22_response(entry['did'], raw)

                else:
                    raise ValueError(f"Unknown entry type: {entry['type']}")

                value = entry['parse'](payload)
                row[entry['name']] = value

            except TimeoutError:
                row[entry['name']] = None
                print(f"timeout: {entry['name']}")
            except Exception as e:
                row[entry['name']] = None
                print(f"error {entry['name']}: {e}")

        # Write to CSV
        with open(CSV_FILE, "a", newline="") as f:
            w = csv.DictWriter(f, fieldnames=["timestamp"] + columns)
            w.writerow(row)

        # Console peek
        pretty = ", ".join(f"{k}={row.get(k)}" for k in columns)
        print(f"{row['timestamp']}  {pretty}")

        time.sleep(POLL_PERIOD_S)

if __name__ == "__main__":
    main()
