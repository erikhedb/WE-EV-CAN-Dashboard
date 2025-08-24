# Dynamic CAN logger: listen to can0, store latest message for all CAN IDs if CRC changes
import can
import time
import threading
import json
import os
import datetime
from handlers import handlers

# This is two dictionaries to store the latest message and last CRC for each CAN ID
latest_messages = {}
last_crc = {}

def save_and_print_messages_periodically():
    while True:
        time.sleep(5)
        out = []
        for can_id, msg in latest_messages.items():
            handler = handlers.get(can_id, handlers['default'])
            formatted = handler(bytes.fromhex(msg['data']), can_id)
            # Convert timestamp to ISO8601 string
            formatted["timestamp"] = datetime.datetime.fromtimestamp(formatted["timestamp"]).isoformat()
            hex_id = formatted["can_id"]
            fname = os.path.join("data", f"{hex_id}.json")
            os.makedirs("data", exist_ok=True)
            with open(fname, "w") as f:
                json.dump(formatted, f, indent=2)
            out.append(formatted)
        # Optionally print all formatted messages
        # print(json.dumps(out, indent=2))

def main():
    bus = can.interface.Bus(channel='can0', bustype='socketcan')
    printer_thread = threading.Thread(target=save_and_print_messages_periodically, daemon=True)
    printer_thread.start()
    try:
        for msg in bus:
            can_id = msg.arbitration_id
            data = bytes(msg.data)
            if len(data) >= 8:
                crc = data[7]
                if last_crc.get(can_id) != crc:
                    # Store raw message only
                    latest_messages[can_id] = {
                        "can_id": can_id,
                        "data": data.hex(),
                        "crc": crc,
                        "timestamp": time.time()
                    }
                    last_crc[can_id] = crc
            # else: ignore messages with less than 8 bytes
    except KeyboardInterrupt:
        pass

if __name__ == "__main__":
    main()