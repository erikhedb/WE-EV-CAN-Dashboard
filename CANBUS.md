# CANBUS Messages


## ORION BMS 2

0x36 - Battery Cell brodcast Id
0x76 - Thermistor Brodcast Id


Bit Order - Most siginificant bit first -> 00000001b = 1 in decimal 
Byte Order - Big Endian = Most significant byte first -> 0001h = 1 in decimal

```json
{"id":"6B0","length":8,"data":"00A100486E50005F","meta":39}
{"id":"6B1","length":8,"data":"00120000993A009E","meta":0}
{"id":"6B4","length":8,"data":"0162000412000000","meta":0}
{"id":"6B3","length":8,"data":"0013000000100000","meta":0}
{"id":"6B2","length":8,"data":"00460000990100E0","meta":0}
```

0x6B0 - 00A100486E50005F
    SOC             -> 00A1 = 161    -> 80.5 ( * 10 / 2)
    Cell Count      -> 0048 = 72     -> 72     
    Pack Voltage    -> 6E50 = 28.240 -> 282.4 ( * 10)
    00              -> Not in use
    5F              -> Checksum - Algo ?


0x6B1 - 0012 0000 993A 009E
    High Cell Id        -> 0012 = 18 -> 18
    High Cell Voltage   -> 993A = 39,226 -> 3.9336 (/ 10)

0x6B2 - 0046 0000 9901 00E0
    Low Cell Id        -> 0046 = 70 -> 70
    Low Cell Voltage   -> 9901 = 39.169 -> 3.9169 (/ 10)

0x6B3 - 0013 0000 0010 0000
    High Temp   -> 0013 = 19    -> 19
    Low Temp    -> 0010 = 16    -> 16

0x6B4 - 0162 0004 1200 0000
    Relay State -> 0001 0110 0010 
    Pack CCL    -> 0004 = 4         -> 4h
    Pack DCL    -> 1200 = 4608      -> ???


## Zero EV CSS Controller - From the Orion BMS 2

0x357
0x35A
0x356
0x355
0x351
0x35B


## Total Raw frames in sample from 2025-09-16
ID     Count
----------------
036    52   - Battery Cell brodcast
076    262  - Thermistor Brodcast Id
351    252  - From BMS to EV Controller
355    252  - From BMS to EV Controller
356    252  - From BMS to EV Controller
35A    52   - From BMS to EV Controller
35B    52   - From BMS to EV Controller
374    37
375    38
376    38
377    38
379    37
380    37
381    37
6B0    26   - Custom frame
6B1    26   - Custom frame
6B2    26   - Custom frame
6B3    26   - Custom frame
6B4    26   - Custom frame