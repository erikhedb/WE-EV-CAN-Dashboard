#!/usr/bin/env python3
"""
Convert a candump log (-L format) to a PCAN/Vector-compatible TRC v2 file.
Usage:
    python to_pcan.py input.log > output.trc
    python to_pcan.py input.log --output output.trc
"""

import argparse
import datetime
import pathlib
import re
import sys
from typing import Iterable, Iterator, Optional, TextIO, Tuple

# Example candump line:
# (1702392714.875963)  can0  00000301#1122334455667788
LINE_RE = re.compile(r"\((\d+\.\d+)\)\s+(\S+)\s+([0-9A-Fa-f]+)#([0-9A-Fa-f]*)")

Parsed = Tuple[float, str, int, Tuple[str, ...]]


def parse_line(line: str) -> Optional[Parsed]:
    match = LINE_RE.match(line.strip())
    if not match:
        return None
    ts, iface, can_id_hex, data_hex = match.groups()
    data = tuple(data_hex[i : i + 2] for i in range(0, len(data_hex), 2))
    return float(ts), iface, int(can_id_hex, 16), data


def header(start_ts: float) -> str:
    start_dt = datetime.datetime.fromtimestamp(start_ts)
    return (
        ";$FILEVERSION=2.0\n"
        f";$STARTTIME={start_dt:%Y-%m-%d %H:%M:%S.%f}\n"
        ";$COLUMNS=N,O,T,I,d,L,B1,B2,B3,B4,B5,B6,B7,B8\n"
    )


def format_row(idx: int, t_rel: float, iface: str, can_id: int, data: Tuple[str, ...]) -> str:
    dlc = len(data)
    # TRC expects exactly 8 byte columns; pad with "--" if fewer.
    bytes_str = " ".join((*data, *("--" for _ in range(max(0, 8 - dlc)))))
    return f"{idx:6d}) {t_rel:10.6f} {iface:>5} Rx {can_id:08X} {dlc:1d} {bytes_str}\n"


def to_trc(rows: Iterable[str]) -> Iterator[str]:
    idx = 0
    t0: Optional[float] = None
    for raw in rows:
        parsed = parse_line(raw)
        if not parsed:
            continue

        ts, iface, can_id, data = parsed
        if t0 is None:
            t0 = ts
            yield header(t0)

        idx += 1
        yield format_row(idx, ts - t0, iface, can_id, data)


def convert(src: pathlib.Path, dst: TextIO) -> None:
    with src.open("r", encoding="utf-8") as handle:
        wrote = False
        for chunk in to_trc(handle):
            dst.write(chunk)
            wrote = True
        if not wrote:
            raise SystemExit("no parsable candump lines found")


def main(argv: Optional[Iterable[str]] = None) -> int:
    parser = argparse.ArgumentParser(description="Convert candump -L logs to TRC v2.")
    parser.add_argument("input", type=pathlib.Path, help="candump log file")
    parser.add_argument(
        "-o", "--output", type=pathlib.Path, help="output TRC file (defaults to stdout)"
    )
    args = parser.parse_args(argv)

    if args.output:
        with args.output.open("w", encoding="utf-8") as dst:
            convert(args.input, dst)
    else:
        convert(args.input, sys.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
