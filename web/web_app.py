#!/usr/bin/env python3
import os, json, argparse, datetime
from typing import Any, Dict, List
from flask import Flask, render_template, abort

APP_DIR = os.path.abspath(os.path.dirname(__file__))
DEFAULT_JSON = os.path.join(APP_DIR, "..", "logger", "data", "bms_data.json")
TEMPLATE_NAME = "dashboard.html"  # must sit next to this app.py

def _read_json(path: str) -> tuple[Dict[str, Any], str]:
    try:
        with open(path, "r", encoding="utf-8") as f:
            text = f.read()
        return json.loads(text), text
    except Exception as e:
        raise RuntimeError(f"Failed to read JSON {path}: {e}")

def _epoch_to_local(ts: float | None) -> str:
    if not ts:
        return "—"
    try:
        dt = datetime.datetime.fromtimestamp(ts)
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    except Exception:
        return "—"

def _safe_num(x, nd=None):
    if x is None:
        return None
    try:
        return round(float(x), nd) if nd is not None else float(x)
    except Exception:
        return None

def build_context(data: Dict[str, Any], raw_text: str, json_path: str) -> Dict[str, Any]:
    # Map frames by CAN ID
    frames = {f.get("can_id"): f for f in data.get("frames", []) if isinstance(f, dict)}

    f6B0 = frames.get("6B0", {})  # pack current/voltage/SOC/relay
    f6B1 = frames.get("6B1", {})  # pack DCL/CCL/temperatures
    f6B2 = frames.get("6B2", {})  # cell voltages and IDs

    # Pack metrics
    pack = {
        "current_A": _safe_num(f6B0.get("Pack_Current"), 2),
        "voltage_V": _safe_num(f6B0.get("Pack_Inst_Voltage"), 2),
        "soc_pct": _safe_num(f6B0.get("Pack_SOC"), 1),
        "relay_word": f6B0.get("relay_state_word"),
        "relay_flags": f6B0.get("relay_flags_lowbyte") or {},
        "crc_ok": bool(f6B0.get("crc_ok")),
    }

    # Pack DCL/CCL and temperatures
    pack_limits = {
        "dcl_A": _safe_num(f6B1.get("Pack_DCL"), 2),
        "ccl_A": _safe_num(f6B1.get("Pack_CCL"), 2),
        "high_temp_C": _safe_num(f6B1.get("High_Temperature"), 1),
        "low_temp_C": _safe_num(f6B1.get("Low_Temperature"), 1),
        "crc_ok": bool(f6B1.get("crc_ok")),
    }

    # Cell voltages and IDs
    low_cell_voltage = _safe_num(f6B2.get("Low_Cell_Voltage"), 4)
    high_cell_delta = _safe_num(f6B2.get("High_Cell_Voltage"), 4)
    
    # Calculate high cell voltage (low + delta)
    high_cell_voltage = None
    if low_cell_voltage is not None and high_cell_delta is not None:
        high_cell_voltage = low_cell_voltage + high_cell_delta
        print(f"DEBUG: Low cell: {low_cell_voltage}, Delta: {high_cell_delta}, High cell: {high_cell_voltage}")
    
    cell_data = {
        "low_cell_voltage_V": low_cell_voltage,
        "high_cell_delta_V": high_cell_delta,
        "high_cell_voltage_V": high_cell_voltage,
        "high_cell_id": f6B2.get("High_Cell_ID"),
        "low_cell_id": f6B2.get("Low_Cell_ID"),
        "crc_ok": bool(f6B2.get("crc_ok")),
    }

    # File metadata
    try:
        mtime = os.path.getmtime(json_path)
        mtime_str = _epoch_to_local(mtime)
    except Exception:
        mtime_str = "—"

    written_at_str = _epoch_to_local(data.get("written_at_epoch"))
    mode = data.get("mode", "—")

    return {
        "json_path": os.path.abspath(json_path),
        "file_mtime": mtime_str,
        "written_at": written_at_str,
        "mode": mode,
        "raw_json": raw_text,
        "pack": pack,
        "pack_limits": pack_limits,
        "cell_data": cell_data,
    }

def create_app(json_path: str) -> Flask:
    # Point template_folder to the same directory so the html file can live next to app.py
    app = Flask(__name__, template_folder=APP_DIR)

    @app.route("/")
    def index():
        try:
            data, raw = _read_json(json_path)
            ctx = build_context(data, raw, json_path)
            return render_template(TEMPLATE_NAME, **ctx)
        except Exception as e:
            return abort(500, description=str(e))

    return app

def parse_args():
    ap = argparse.ArgumentParser(description="Simple Flask dashboard for Orion BMS JSON")
    ap.add_argument("--json", default=DEFAULT_JSON, help="Path to bms_data.json (default: ./bms_data.json)")
    ap.add_argument("--host", default="127.0.0.1", help="Bind host (default: 127.0.0.1)")
    ap.add_argument("--port", type=int, default=8080, help="Bind port (default: 8080)")
    return ap.parse_args()

if __name__ == "__main__":
    args = parse_args()
    app = create_app(args.json)
    app.run(host=args.host, port=args.port, debug=False)
