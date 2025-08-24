#!/usr/bin/env python3
"""
Orion BMS v2 CAN logger (live or offline)
- Decodes 0x6B0, 0x6B1, 0x6B2
- Passively reassembles UDS (ISO-TP) from 0x7EB for cell voltages (F1 00..F1 05)
- Writes JSON every N seconds in live mode, or once in offline mode

Usage (live CAN):
  sudo python3 orion_logger.py --out bms_data.json --interval 10

Usage (offline log):
  python3 orion_logger.py --can-file can0.log --out bms_data.json
  # alias also accepted:
  python3 orion_logger.py --canfile can0.log --out bms_data.json

Requirements (live):
  python-can, socketcan interface up (e.g. 500k):
    sudo apt install python3-can can-utils
    sudo ip link set can0 up type can bitrate 500000

Log format supported (candump style):
  (timestamp)  IFACE   ID   [DLC]  bytes...
  example: (1755680987.163825)  can0       7E3   [8]  03 22 F2 02 00 00 00 00
"""
from __future__ import annotations
import argparse
import json
import os
import signal
import sys
import time
from typing import Any, Dict, List, Optional, Tuple

# ------------------------ helpers ------------------------

def _u16_be(b: bytes) -> int:
    return (b[0] << 8) | b[1]


def _hex_bytes(b: bytes) -> str:
    return " ".join(f"{x:02X}" for x in b)


def _checksum8_sum(data: bytes) -> int:
    """Simple 8-bit sum over data[0:7] compared to data[7]. Heuristic only."""
    if len(data) < 8:
        return -1
    return (sum(data[:7]) & 0xFF)

# ------------------------ relay mapping (low byte only) ------------------------

def relay_lowbyte_flags(low: int) -> Dict[str, bool | str]:
    return {
        "discharge_enabled": bool(low & 0x01),      # bit 0
        "charge_enabled": bool(low & 0x02),         # bit 1
        "charger_safety_enabled": bool(low & 0x04), # bit 2
        "malfunction_active": bool(low & 0x08),     # bit 3
        "mpi_active": bool(low & 0x10),             # bit 4
        "always_on_active": bool(low & 0x20),       # bit 5
        "is_ready": bool(low & 0x40),               # bit 6
        "is_charging": bool(low & 0x80),            # bit 7
        "raw_low_byte_hex": f"0x{low:02X}",
    }

# ------------------------ decoders for 6B0/6B1/6B2 ------------------------

def decode_6B0(data: bytes, limits: argparse.Namespace) -> Dict[str, Any]:
    #  Pack_Current       : bytes 0..1 (0.1 A/LSB) [unsigned here]
    #  Pack_Inst_Voltage  : bytes 2..3 (0.1 V/LSB)
    #  Pack_SOC           : byte 4 (0.5 %/LSB)
    #  Relay_State        : bytes 5..6 (word)
    #  CRC_Checksum       : byte 7
    pack_current_A = _u16_be(data[0:2]) * 0.1
    if not (limits.pack_i_min <= pack_current_A <= limits.pack_i_max):
        pack_current_A = None

    pack_inst_voltage_V = _u16_be(data[2:4]) * 0.1
    if not (limits.pack_v_min <= pack_inst_voltage_V <= limits.pack_v_max):
        pack_inst_voltage_V = None

    pack_soc_percent = data[4] * 0.5
    if not (limits.soc_min <= pack_soc_percent <= limits.soc_max):
        pack_soc_percent = None

    relay_state_word = (data[5] << 8) | data[6]
    relay_flags = relay_lowbyte_flags(relay_state_word & 0xFF)

    crc = data[7]
    crc_ok = (_checksum8_sum(data) == crc)

    return {
        "can_id": "6B0",
        "raw": _hex_bytes(data),
        "pack_current_A": pack_current_A,
        "pack_inst_voltage_V": pack_inst_voltage_V,
        "pack_soc_percent": pack_soc_percent,
        "relay_state_word": f"0x{relay_state_word:04X}",
        "relay_flags_lowbyte": relay_flags,
        "crc": f"0x{crc:02X}",
        "crc_ok": crc_ok,
    }


def decode_6B1(data: bytes, limits: argparse.Namespace) -> Dict[str, Any]:
    #  Pack_DCL           : bytes 0..1 (1.0 A/LSB)
    #  Pack_CCL           : byte 2     (1.0 A/LSB)
    #  Blank              : byte 3
    #  High_Temperature   : byte 4     (°C)
    #  Low_Temperature    : byte 5     (°C)
    #  Blank              : byte 6
    #  CRC_Checksum       : byte 7
    pack_dcl_A = _u16_be(data[0:2]) * 1.0
    pack_ccl_A = data[2] * 1.0

    high_temperature_C = float(data[4])
    low_temperature_C = float(data[5])

    # range filter
    if not (limits.temp_min <= high_temperature_C <= limits.temp_max):
        high_temperature_C = None
    if not (limits.temp_min <= low_temperature_C <= limits.temp_max):
        low_temperature_C = None

    crc = data[7]
    crc_ok = (_checksum8_sum(data) == crc)

    return {
        "can_id": "6B1",
        "raw": _hex_bytes(data),
        "pack_dcl_A": pack_dcl_A,
        "pack_ccl_A": pack_ccl_A,
        "high_temperature_C": high_temperature_C,
        "low_temperature_C": low_temperature_C,
        "crc": f"0x{crc:02X}",
        "crc_ok": crc_ok,
    }


def decode_6B2(data: bytes, limits: argparse.Namespace) -> Dict[str, Any]:
    #  High_Cell_Voltage    : bytes 0..1 (1e-4 V/LSB)
    #  Low_Cell_Voltage     : bytes 2..3 (1e-4 V/LSB)
    #  High_Cell_Voltage_ID : byte 4
    #  Low_Cell_Voltage_ID  : byte 5
    #  Populated_Cells      : byte 6
    #  CRC_Checksum         : byte 7
    high_cell_voltage_V = _u16_be(data[0:2]) * 1e-4
    low_cell_voltage_V = _u16_be(data[2:4]) * 1e-4
    if not (limits.cell_min <= high_cell_voltage_V <= limits.cell_max):
        high_cell_voltage_V = None
    if not (limits.cell_min <= low_cell_voltage_V <= limits.cell_max):
        low_cell_voltage_V = None

    high_cell_id = int(data[4])
    low_cell_id = int(data[5])
    populated_cells = int(data[6])

    crc = data[7]
    crc_ok = (_checksum8_sum(data) == crc)

    return {
        "can_id": "6B2",
        "raw": _hex_bytes(data),
        "high_cell_voltage_V": high_cell_voltage_V,
        "low_cell_voltage_V": low_cell_voltage_V,
        "high_cell_id": high_cell_id,
        "low_cell_id": low_cell_id,
        "populated_cells": populated_cells,
        "crc": f"0x{crc:02X}",
        "crc_ok": crc_ok,
    }


DECODERS = {
    0x6B0: decode_6B0,
    0x6B1: decode_6B1,
    0x6B2: decode_6B2,
}

# ------------------------ ISO-TP collector for 0x7EB ------------------------

class IsoTpCollector:
    """Passive ISO-TP collector for responses from 0x7EB."""
    def __init__(self):
        self.active: Optional[Dict[str, Any]] = None

    def ingest(self, can_id: int, data: bytes) -> Optional[bytes]:
        if can_id != 0x7EB or not data:
            return None
        pci_type = data[0] & 0xF0
        if pci_type == 0x10:  # First Frame
            total = ((data[0] & 0x0F) << 8) | data[1]
            self.active = {"total": total, "buf": bytearray(data[2:]), "next_sn": 1}
            if len(self.active["buf"]) >= total:
                return self._finish()
            return None
        if pci_type == 0x20 and self.active:  # Consecutive Frame
            sn = data[0] & 0x0F
            if sn != self.active["next_sn"]:
                self.active = None
                return None
            self.active["buf"].extend(data[1:])
            self.active["next_sn"] = (self.active["next_sn"] + 1) & 0x0F
            if len(self.active["buf"]) >= self.active["total"]:
                return self._finish()
            return None
        if pci_type == 0x00:  # Single Frame (rare in your dump)
            sf_len = data[0] & 0x0F
            payload = data[1:1 + sf_len]
            return bytes(payload)
        return None

    def _finish(self) -> bytes:
        buf = bytes(self.active["buf"][: self.active["total"]])
        self.active = None
        return buf


# ------------------------ UDS F1xx parser (72 cells) ------------------------

def parse_uds_positive(buf: bytes, cell_voltages: List[Optional[float]],
                       cell_min: float, cell_max: float) -> bool:
    """Parse a positive UDS response (0x62 ...). Returns True if any cells were filled."""
    if not buf or buf[0] != 0x62 or len(buf) < 3:
        return False
    did = (buf[1] << 8) | buf[2]
    data = buf[3:]
    # F1 00..F1 05: 12 big-endian words, 0.0001 V/LSB
    if 0xF100 <= did <= 0xF105 and len(data) >= 24:
        group_idx = did - 0xF100  # 0..5
        for i in range(12):
            raw = (data[2 * i] << 8) | data[2 * i + 1]
            v = raw * 1e-4
            cell_idx = group_idx * 12 + i  # 0..71
            cell_voltages[cell_idx] = v if (cell_min <= v <= cell_max) else None
        return True
    return False


# ------------------------ building JSON blocks ------------------------

def build_frames_out(last: Dict[int, Dict[str, Any]]) -> List[Dict[str, Any]]:
    out: List[Dict[str, Any]] = []
    for can_id in (0x6B0, 0x6B1, 0x6B2):
        if can_id in last:
            out.append(last[can_id])
    return out


def build_cell_groups(cells: List[Optional[float]]) -> List[Dict[str, Any]]:
    groups = []
    for g in range(6):
        start = g * 12
        block = cells[start:start + 12]
        groups.append({
            "module": g + 1,
            "cells": [
                {"index": start + i + 1, "voltage_V": block[i]} for i in range(12)
            ],
        })
    return groups


def build_derived(cells: List[Optional[float]]) -> Dict[str, Any]:
    vals = [v for v in cells if v is not None]
    if not vals:
        return {
            "cells_valid": 0,
            "min_V": None,
            "max_V": None,
            "avg_V": None,
            "delta_V": None,
        }
    mn = min(vals)
    mx = max(vals)
    avg = sum(vals) / len(vals)
    return {
        "cells_valid": len(vals),
        "min_V": mn,
        "max_V": mx,
        "avg_V": avg,
        "delta_V": mx - mn,
    }


# ------------------------ candump log parser ------------------------

def parse_log_line(line: str) -> Optional[Tuple[int, bytes]]:
    """Very forgiving parser for lines like:
    (1755680987.163825)  can0  6B1  [8]  00 00 4F 00 26 15 48 01
    """
    # Find the closing paren after timestamp
    try:
        rparen = line.index(')')
    except ValueError:
        return None
    rest = line[rparen + 1:].strip()
    if not rest:
        return None
    parts = rest.split()
    if len(parts) < 4:
        return None
    # parts[0]=iface, parts[1]=id, parts[2]=[dlc], parts[3:]=bytes
    id_str = parts[1]
    try:
        can_id = int(id_str, 16)
    except ValueError:
        return None
    data_bytes: List[int] = []
    for tok in parts[3:]:
        tok = tok.strip('[]')
        if len(tok) != 2:
            continue
        try:
            data_bytes.append(int(tok, 16))
        except ValueError:
            continue
    if not data_bytes:
        return None
    return can_id, bytes(data_bytes)


# ------------------------ main ------------------------

def process_can_file(path: str, args: argparse.Namespace) -> Dict[str, Any]:
    last_messages: Dict[int, Dict[str, Any]] = {}
    cell_voltages: List[Optional[float]] = [None] * 72
    cell_updated_ts: Optional[float] = None
    iso = IsoTpCollector()

    with open(path, "r", encoding="utf-8", errors="ignore") as f:
        for line in f:
            parsed = parse_log_line(line)
            if not parsed:
                continue
            can_id, data = parsed

            # Base frames
            if can_id in DECODERS and len(data) >= 8:
                try:
                    last_messages[can_id] = DECODERS[can_id](data, args)
                except Exception as e:
                    print(f"Decode error for ID 0x{can_id:X}: {e}", file=sys.stderr)

            # ISO-TP for 0x7EB
            assembled = iso.ingest(can_id, data)
            if assembled:
                if parse_uds_positive(assembled, cell_voltages, args.cell_min, args.cell_max):
                    cell_updated_ts = time.time()

    payload = {
        "frames": build_frames_out(last_messages),
        "cells": {
            "groups": build_cell_groups(cell_voltages),
            "derived": build_derived(cell_voltages),
            "last_update_epoch": cell_updated_ts,
        },
        "written_at_epoch": time.time(),
        "mode": "offline",
        "source_file": os.path.abspath(path),
    }
    return payload


def run_live(args: argparse.Namespace) -> None:
    # Lazy import python-can so offline mode works without it
    try:
        import can  # type: ignore
    except ImportError:
        print("python-can is required for live capture. Install with: pip install python-can", file=sys.stderr)
        sys.exit(1)

    bus = can.interface.Bus(channel=args.channel, bustype=args.bustype)

    last_messages: Dict[int, Dict[str, Any]] = {}
    cell_voltages: List[Optional[float]] = [None] * 72
    cell_updated_ts: Optional[float] = None
    iso = IsoTpCollector()

    running = True
    last_write = time.monotonic()

    def handle_sigint(sig, frame):
        nonlocal running
        running = False

    signal.signal(signal.SIGINT, handle_sigint)
    signal.signal(signal.SIGTERM, handle_sigint)

    try:
        while running:
            msg = bus.recv(timeout=1.0)
            if msg is None:
                pass
            else:
                arb = msg.arbitration_id
                data = bytes(msg.data)

                if arb in DECODERS and len(data) >= 8:
                    try:
                        last_messages[arb] = DECODERS[arb](data, args)
                    except Exception as e:
                        print(f"Decode error for ID 0x{arb:X}: {e}", file=sys.stderr)

                assembled = iso.ingest(arb, data)
                if assembled:
                    if parse_uds_positive(assembled, cell_voltages, args.cell_min, args.cell_max):
                        cell_updated_ts = time.time()

            now = time.monotonic()
            if now - last_write >= args.interval:
                payload = {
                    "frames": build_frames_out(last_messages),
                    "cells": {
                        "groups": build_cell_groups(cell_voltages),
                        "derived": build_derived(cell_voltages),
                        "last_update_epoch": cell_updated_ts,
                    },
                    "written_at_epoch": time.time(),
                    "mode": "live",
                }
                try:
                    with open(args.out, "w", encoding="utf-8") as f:
                        json.dump(payload, f, indent=2)
                    
                except Exception as e:
                    print(f"Failed to write JSON: {e}", file=sys.stderr)
                last_write = now
    finally:
        # Final write on exit
        payload = {
            "frames": build_frames_out(last_messages),
            "cells": {
                "groups": build_cell_groups(cell_voltages),
                "derived": build_derived(cell_voltages),
                "last_update_epoch": cell_updated_ts,
            },
            "written_at_epoch": time.time(),
            "mode": "live",
        }
        try:
            with open(args.out, "w", encoding="utf-8") as f:
                json.dump(payload, f, indent=2)
        except Exception as e:
            print(f"Failed to write JSON on exit: {e}", file=sys.stderr)


def build_argparser() -> argparse.ArgumentParser:
    ap = argparse.ArgumentParser(description="Orion BMS v2 CAN logger (live or offline)")
    ap.add_argument("--channel", default="can0", help="CAN interface (live), default can0")
    ap.add_argument("--bustype", default="socketcan", help="python-can bustype, default socketcan")
    ap.add_argument("--out", default="bms_data.json", help="Output JSON file path (default bms_data.json)")
    ap.add_argument("--interval", type=float, default=10.0, help="Write interval seconds in live mode (default 10)")

    # offline log support (accept both spellings)
    ap.add_argument("--can-file", "--canfile", dest="can_file", help="Path to a candump-style .log file for offline parse")

    # range filters
    ap.add_argument("--cell-min", type=float, default=2.0, help="Min valid cell voltage (V)")
    ap.add_argument("--cell-max", type=float, default=4.5, help="Max valid cell voltage (V)")
    ap.add_argument("--temp-min", type=float, default=-40.0, help="Min valid temperature (°C)")
    ap.add_argument("--temp-max", type=float, default=120.0, help="Max valid temperature (°C)")
    ap.add_argument("--pack-v-min", type=float, default=0.0, help="Min valid pack voltage (V)")
    ap.add_argument("--pack-v-max", type=float, default=400.0, help="Max valid pack voltage (V)")
    ap.add_argument("--pack-i-min", type=float, default=0.0, help="Min valid pack current (A)")
    ap.add_argument("--pack-i-max", type=float, default=2000.0, help="Max valid pack current (A)")
    ap.add_argument("--soc-min", type=float, default=0.0, help="Min valid SOC (%)")
    ap.add_argument("--soc-max", type=float, default=100.0, help="Max valid SOC (%)")

    return ap


def main() -> None:
    ap = build_argparser()
    args = ap.parse_args()

    if args.can_file:
        payload = process_can_file(args.can_file, args)
        try:
            with open(args.out, "w", encoding="utf-8") as f:
                json.dump(payload, f, indent=2)
        except Exception as e:
            print(f"Failed to write JSON: {e}", file=sys.stderr)
        return

    # live mode
    run_live(args)


if __name__ == "__main__":
    main()
