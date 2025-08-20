#!/usr/bin/env python3
"""
Count CAN frames per ID from a standard candump log and sort by count.

Usage:
  python3 count_can_ids.py candump.log
  # or read from stdin:
  candump can0 | python3 count_can_ids.py -

Options:
  -n / --top N   show top N (default: all)
  --csv out.csv  also write results to CSV
"""
import sys, re, argparse, csv
from collections import Counter

# candump "standard" format:
# (TS) IFACE ID [DLC] DATA...
RE_STD = re.compile(
    r"""^\s*
        (?:\((?P<ts>[\d\.]+)\)\s*)?      # optional (timestamp)
        (?P<iface>\w+)\s+                # can0 / can1
        (?P<id>[0-9A-Fa-f]+)\s+          # CAN ID (3 or 8 hex)
        \[(?P<dlc>\d+)\]\s+              # [DLC]
        (?P<data>(?:[0-9A-Fa-f]{2}\s*)*) # bytes (optional)
        """,
    re.VERBOSE,
)

# compact form support: "can0 123#DEADBEEF"
RE_COMPACT = re.compile(
    r"""^\s*
        (?P<iface>\w+)\s+
        (?P<id>[0-9A-Fa-f]+)
        \#
        (?P<data>[0-9A-Fa-f]*)
        """,
    re.VERBOSE,
)

def parse_line(line: str):
    m = RE_STD.match(line)
    if m:
        return m.group("iface"), m.group("id").upper()
    m = RE_COMPACT.match(line)
    if m:
        return m.group("iface"), m.group("id").upper()
    return None

def main():
    ap = argparse.ArgumentParser(description="Count CAN IDs in candump logs")
    ap.add_argument("path", help="candump log file or '-' for stdin")
    ap.add_argument("-n", "--top", type=int, default=None, help="show top N")
    ap.add_argument("--csv", help="optional CSV output path")
    args = ap.parse_args()

    counts = Counter()
    per_iface = Counter()

    # Input stream
    it = sys.stdin if args.path == "-" else open(args.path, "r", encoding="utf-8", errors="ignore")
    try:
        for line in it:
            parsed = parse_line(line)
            if not parsed:
                continue
            iface, can_id = parsed
            counts[can_id] += 1
            per_iface[iface] += 1
    finally:
        if it is not sys.stdin:
            it.close()

    # Sorted results
    items = sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))
    if args.top is not None:
        items = items[:args.top]

    total = sum(counts.values()) or 1
    print(f"# Frames total: {total}")
    if per_iface:
        print("# By interface: " + ", ".join(f"{k}={v}" for k, v in per_iface.items()))
    print("#")
    print(f"{'Rank':>4}  {'CAN_ID':<8}  {'Count':>8}  {'Percent':>8}")
    print("-"*36)
    for i, (cid, cnt) in enumerate(items, 1):
        pct = 100.0 * cnt / total
        print(f"{i:>4}  {cid:<8}  {cnt:>8}  {pct:>7.2f}%")

    # Optional CSV
    if args.csv:
        with open(args.csv, "w", newline="", encoding="utf-8") as f:
            w = csv.writer(f)
            w.writerow(["rank", "can_id", "count", "percent"])
            for i, (cid, cnt) in enumerate(sorted(counts.items(), key=lambda kv: (-kv[1], kv[0])), 1):
                pct = 100.0 * cnt / total
                w.writerow([i, cid, cnt, f"{pct:.4f}"])
        print(f"\nWrote CSV: {args.csv}")

if __name__ == "__main__":
    main()
