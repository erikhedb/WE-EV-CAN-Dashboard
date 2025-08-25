# CAN UDS cell voltage poller for Orion BMS (72S)
# Triggers a request every 5 seconds, assembles cell voltage data, prints summary and response counts
import can
import time
import threading

# UDS request blocks for cell voltages (F1 00 to F1 05)
REQUEST_IDS = [0xF100, 0xF101, 0xF102, 0xF103, 0xF104, 0xF105]

class CellVoltageCollector:
    def __init__(self):
        self.responses = {rid: [] for rid in REQUEST_IDS}
        self.counts = {rid: 0 for rid in REQUEST_IDS}
        self.cell_voltages = [None] * 72
        self.lock = threading.Lock()

    def ingest(self, can_id, data):
        # Only process 0x7EB responses
        if can_id != 0x7EB:
            return
        pci = data[0]
        if pci & 0xF0 == 0x10:  # First frame
            # Extract DID
            did = (data[3] << 8) | data[4]
            self.responses[did] = data[5:]
            self.counts[did] += 1
        elif pci & 0xF0 in (0x20, 0x00):  # Consecutive or single frame
            # Find last DID (from previous first frame)
            for did in REQUEST_IDS:
                if self.responses[did]:
                    self.responses[did] += data[1:]
                    break
        # Try to parse voltages if enough data
        for did in REQUEST_IDS:
            if len(self.responses[did]) >= 24:
                self._update_block(did, self.responses[did][:24])
                self.responses[did] = []

    def _update_block(self, did, block):
        group = did - 0xF100
        for i in range(12):
            raw = (block[2*i] << 8) | block[2*i+1]
            v = round(raw * 1e-4, 3)
            idx = group * 12 + i
            # Only update if voltage is in a reasonable range
            if 3.5 <= v <= 4.2:
                with self.lock:
                    if v != self.cell_voltages[idx]:
                        self.cell_voltages[idx] = v

    def get_summary(self):
        with self.lock:
            valid = [v for v in self.cell_voltages if v is not None]
            return {
                "cells_valid": len(valid),
                "min_V": round(min(valid), 3) if valid else None,
                "max_V": round(max(valid), 3) if valid else None,
                "avg_V": round(sum(valid)/len(valid), 3) if valid else None,
                "delta_V": round((max(valid)-min(valid)), 3) if valid else None,
                "cell_voltages": self.cell_voltages.copy(),
                "response_counts": self.counts.copy()
            }


def send_requests(bus):
    # Send all 6 requests for cell voltages
    for did in REQUEST_IDS:
        # UDS ReadDataByIdentifier: 03 22 <DID> 00 00 00 00 00
        data = [0x03, 0x22, (did >> 8) & 0xFF, did & 0xFF, 0, 0, 0, 0]
        msg = can.Message(arbitration_id=0x7E3, data=bytes(data), is_extended_id=False)
        bus.send(msg)


def poller():
    bus = can.interface.Bus(channel='can0', bustype='socketcan')
    collector = CellVoltageCollector()
    def request_loop():
        while True:
            send_requests(bus)
            time.sleep(5)
    threading.Thread(target=request_loop, daemon=True).start()
    print("Polling cell voltages every 5 seconds...")
    try:
        while True:
            msg = bus.recv(timeout=1.0)
            if msg is None:
                continue
            can_id = msg.arbitration_id
            data = list(msg.data)
            if can_id == 0x7EB:
                collector.ingest(can_id, data)
            # Print incoming response counts for 0x7EB
            if can_id == 0x7EB:
                print(f"Received 0x7EB: {data}")
            # Optionally print other E-type frames
            elif can_id in (0x7DE, 0x7E3):
                print(f"Received 0x{can_id:X}: {data}")
            # Print summary every 5 seconds
            if int(time.time()) % 5 == 0:
                summary = collector.get_summary()
                print(f"Valid:{summary['cells_valid']} Min:{summary['min_V']} Max:{summary['max_V']} Avg:{summary['avg_V']} Δ:{summary['delta_V']}")
                # Print 72S voltages in rows of 12
                cells = summary['cell_voltages']
                for row in range(6):
                    row_vals = cells[row*12:(row+1)*12]
                    print(' '.join(f"{v if v is not None else '-':>5}" for v in row_vals))
    except KeyboardInterrupt:
        print("Stopped.")

if __name__ == "__main__":
    poller()
