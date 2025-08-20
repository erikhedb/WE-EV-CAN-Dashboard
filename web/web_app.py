#!/usr/bin/env python3
import os, json, argparse, datetime
from typing import Any, Dict, List
from flask import Flask, render_template, abort

APP_DIR = os.path.abspath(os.path.dirname(__file__))
DEFAULT_JSON = os.path.join(APP_DIR, "bms_data.json")
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
    f6B1 = frames.get("6B1", {})  # temps
    f6B2 = frames.get("6B2", {})  # hi/lo cell + IDs + populated

    # Pack/temps primary metrics
    pack = {
        "current_A": _safe_num(f6B0.get("pack_current_A"), 2),
        "voltage_V": _safe_num(f6B0.get("pack_inst_voltage_V"), 2),
        "soc_pct": _safe_num(f6B0.get("pack_soc_percent"), 1),
        "relay_word": f6B0.get("relay_state_word"),
        "relay_flags": f6B0.get("relay_flags_lowbyte") or {},
        "crc_ok": bool(f6B0.get("crc_ok")),
    }

    temps = {
        "high_C": _safe_num(f6B1.get("high_temperature_C")),
        "low_C": _safe_num(f6B1.get("low_temperature_C")),
        "crc_ok": bool(f6B1.get("crc_ok")),
    }

    cells_section = data.get("cells", {})
    modules: List[Dict[str, Any]] = []
    derived = {}

    # Expecting your exact structure: cells.groups -> [{module:int, cells:[{index:int, voltage_V:float}, ...]}]
    if isinstance(cells_section, dict) and isinstance(cells_section.get("groups"), list):
        for grp in cells_section["groups"]:
            if not isinstance(grp, dict):
                continue
            mod_no = grp.get("module")
            cell_list = []
            for cell in grp.get("cells", []):
                if not isinstance(cell, dict):
                    continue
                idx = cell.get("index")
                v = _safe_num(cell.get("voltage_V"), 4)
                cell_list.append({"index": idx, "voltage_V": v})
            modules.append({"module": mod_no, "cells": cell_list})

        # derived stats (use provided, fallback to recompute)
        d = cells_section.get("derived") or {}
        vols = [c["voltage_V"] for g in modules for c in g["cells"] if c.get("voltage_V") is not None]
        if vols:
            derived = {
                "cells_valid": int(d.get("cells_valid") or len(vols)),
                "min_V": _safe_num(d.get("min_V") if d.get("min_V") is not None else min(vols), 4),
                "max_V": _safe_num(d.get("max_V") if d.get("max_V") is not None else max(vols), 4),
                "avg_V": _safe_num(d.get("avg_V") if d.get("avg_V") is not None else sum(vols)/len(vols), 4),
                "delta_V": _safe_num(d.get("delta_V") if d.get("delta_V") is not None else (max(vols)-min(vols)), 4),
            }
        else:
            derived = {"cells_valid": 0, "min_V": None, "max_V": None, "avg_V": None, "delta_V": None}
    else:
        # No cells present
        modules = []
        derived = {"cells_valid": 0, "min_V": None, "max_V": None, "avg_V": None, "delta_V": None}

    # 6B2 details (some may be null in your sample)
    cell_ids = {
        "high_cell_voltage_V": _safe_num(f6B2.get("high_cell_voltage_V"), 4),
        "low_cell_voltage_V": _safe_num(f6B2.get("low_cell_voltage_V"), 4),
        "high_cell_id": f6B2.get("high_cell_id"),
        "low_cell_id": f6B2.get("low_cell_id"),
        "populated_cells": f6B2.get("populated_cells"),
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
        "temps": temps,
        "cell_ids": cell_ids,
        "modules": modules,
        "derived": derived,
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
