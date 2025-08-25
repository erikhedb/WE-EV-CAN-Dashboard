# Dump all CAN frames from can0 to a CSV file
import can
import csv
import time

def main():
    bus = can.interface.Bus(channel='can0', bustype='socketcan')
    with open('can_frames.csv', 'w', newline='') as csvfile:
        writer = csv.writer(csvfile)
        writer.writerow(['timestamp', 'can_id', 'dlc', 'data_hex'])
        print("Logging all CAN frames to can_frames.csv...")
        try:
            while True:
                msg = bus.recv(timeout=1.0)
                if msg is None:
                    continue
                timestamp = time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(msg.timestamp))
                can_id = f"0x{msg.arbitration_id:X}"
                dlc = msg.dlc
                data_hex = bytes(msg.data).hex()
                writer.writerow([timestamp, can_id, dlc, data_hex])
        except KeyboardInterrupt:
            print("Stopped logging.")

if __name__ == "__main__":
    main()
