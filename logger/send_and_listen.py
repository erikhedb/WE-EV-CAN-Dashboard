# Send a 0x7DF CAN message and print all 0x7DE, 0x7E3, and 0x7EB responses
import can
import time

def main():
    bus = can.interface.Bus(channel='can0', bustype='socketcan')
    # Construct and send a 0x7DF message (example data: 8 bytes)
    msg = can.Message(arbitration_id=0x7DF, data=bytes.fromhex('02090b0000000000'), is_extended_id=False)
    bus.send(msg)
    print("Sent 0x7DF message.")
    print("Listening for 0x7DE, 0x7E3, 0x7EB responses...")
    try:
        while True:
            rx = bus.recv(timeout=1.0)
            if rx is None:
                continue
            #if rx.arbitration_id in (0x7DE, 0x7E3, 0x7EB):
            print(f"0x{rx.arbitration_id:X}: {bytes(rx.data).hex()}")
    except KeyboardInterrupt:
        print("Stopped listening.")

if __name__ == "__main__":
    main()
