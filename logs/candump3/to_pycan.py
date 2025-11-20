#!/usr/bin/env python3
import sys

if len(sys.argv) < 3:
    print("Usage: python3 convert_to_pcan.py input.txt output.txt")
    sys.exit(1)

infile = sys.argv[1]
outfile = sys.argv[2]

with open(infile, "r") as fin, open(outfile, "w") as fout:
    for line in fin:
        parts = line.strip().split()
        if len(parts) < 4:
            continue  # skip invalid lines

        # candump -L format:
        # iface, canid, [dlc], and then data bytes
        iface = parts[0]
        canid = parts[1]
        dlc = parts[2].strip("[]")
        data_bytes = parts[3:]

        # Join data as space-separated (PCAN format)
        data = " ".join(data_bytes)

        # PCAN-View line (no timestamp in input → use 0.000)
        fout.write(f"0.000  {canid}h  {dlc}  {data}\n")