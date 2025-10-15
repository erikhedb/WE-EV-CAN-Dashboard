# CAN Bus Message Documentation

## Overview

This document describes the CAN bus messages used in the EV system, primarily focusing on communication between the Orion BMS 2 and the Zero EV CSS Controller.

## Data Format Specifications

- **Bit Order**: Most significant bit first (MSB) → `00000001b = 1` decimal
- **Byte Order**: Big Endian (Most significant byte first) → `0001h = 1` decimal

---

## Orion BMS 2 Messages

### Standard Broadcast IDs
- `0x036` - Battery Cell Broadcast ID
- `0x076` - Thermistor Broadcast ID

### Custom Frame Messages

#### Sample Data
```json
{"id":"6B0","length":8,"data":"00A100486E50005F","meta":39}
{"id":"6B1","length":8,"data":"0015FF929BF300ED","meta":0}
{"id":"6B2","length":8,"data":"004600859BAA0010","meta":0}
{"id":"6B3","length":8,"data":"0013000000100000","meta":0}
{"id":"6B4","length":8,"data":"0162000412000000","meta":0}
```

---

### 0x6B0 - Battery Pack Status
**Data**: `00A100486E50005F`

| Bytes | Field | Hex Value | Decimal | Final Value | Calculation | Description |
|-------|-------|-----------|---------|-------------|-------------|-------------|
| 0-1   | SOC | `00A1` | 161 | 80.5% | `161 × 10 ÷ 2` | State of Charge |
| 2-3   | Cell Count | `0048` | 72 | 72 | Direct | Total number of cells |
| 4-5   | Pack Voltage | `6E50` | 28,240 | 282.4V | `28240 ÷ 100` | Pack voltage |
| 6     | Reserved | `00` | 0 | - | - | Not in use |
| 7     | Checksum | `5F` | 95 | - | TBD | Checksum (algorithm unknown) |

---

### 0x6B1 - High Cell Information
**Data**: `0015 FF92 9BF3 00ED`

| Bytes | Field | Hex Value | Decimal | Final Value | Calculation | Description |
|-------|-------|-----------|---------|-------------|-------------|-------------|
| 0-1   | High Cell ID | `0015` | 21 | 21 | Direct | Cell with highest voltage |
| 2-3   | Pack Current | `FF92` | -110 | -11.0A | `Int16(FF92) ÷ 10` | Battery pack current (negative = charging) |
| 4-5   | High Cell Voltage | `9BF3` | 39,923 | 3.9923V | `39923 ÷ 10000` | Voltage of highest cell |
| 6-7   | Reserved | `00ED` | 237 | - | - | Purpose unknown |

Pack current is reported as a signed 16-bit value with 0.1A resolution; positive values indicate discharge and negative values indicate charging.

---

### 0x6B2 - Low Cell Information
**Data**: `0046 0085 9BAA 0010`

| Bytes | Field | Hex Value | Decimal | Final Value | Calculation | Description |
|-------|-------|-----------|---------|-------------|-------------|-------------|
| 0-1   | Low Cell ID | `0046` | 70 | 70 | Direct | Cell with lowest voltage |
| 2-3   | 12V System Voltage | `0085` | 133 | 13.3V | `133 ÷ 10` | Auxiliary system voltage (0.1V resolution) |
| 4-5   | Low Cell Voltage | `9BAA` | 39,850 | 3.9850V | `39850 ÷ 10000` | Voltage of lowest cell |
| 6-7   | Reserved | `0010` | 16 | - | - | Purpose unknown |

The 12V system voltage field reflects the vehicle's auxiliary battery rail rather than the high-voltage pack.

---

### 0x6B3 - Temperature Information
**Data**: `0013 0000 0010 0000`

| Bytes | Field | Hex Value | Decimal | Final Value | Calculation | Description |
|-------|-------|-----------|---------|-------------|-------------|-------------|
| 0-1   | High Temperature | `0013` | 19 | 19°C | Direct | Highest temperature sensor |
| 2-3   | Reserved | `0000` | 0 | - | - | Not used |
| 4-5   | Low Temperature | `0010` | 16 | 16°C | Direct | Lowest temperature sensor |
| 6-7   | Reserved | `0000` | 0 | - | - | Not used |

---

### 0x6B4 - System Control Information
**Data**: `05E6 000B 0A00 0000`

| Bytes | Field | Hex Value | Decimal | Final Value | Calculation | Description |
|-------|-------|-----------|---------|-------------|-------------|-------------|
| 0-1   | Relay State | `05E6` | 1,510 | See below | Bitwise | 16-bit relay/system status |
| 2-3   | Pack CCL | `000B` | 11 | 1.1A | `11 ÷ 10` | Pack Charge Current Limit (0.1A resolution) |
| 4-5   | Pack DCL | `0A00` | 2,560 | 256.0A | `2560 ÷ 10` | Pack Discharge Current Limit (0.1A resolution) |
| 6-7   | Reserved | `0000` | 0 | - | - | Not used |

#### Relay State Bit Mapping (0x05E6 = `0000 0101 1110 0110`)

| Bit | Hex Mask | Decimal | Status | Field Name | Description |
|-----|----------|---------|--------|------------|-------------|
| 0   | `0x01` | 1 | ❌ | Discharge Relay | Discharge relay enabled |
| 1   | `0x02` | 2 | ✅ | Charge Relay | Charge relay enabled |
| 2   | `0x04` | 4 | ✅ | Charger Safety | Charger safety enabled |
| 3   | `0x08` | 8 | ❌ | Malfunction DTC | Malfunction indicator active |
| 4   | `0x10` | 16 | ❌ | MP Input 1 | Multi-Purpose Input signal status |
| 5   | `0x20` | 32 | ✅ | Always On | Always-on signal status |
| 6   | `0x40` | 64 | ✅ | Is Ready | Is-Ready signal status |
| 7   | `0x80` | 128 | ✅ | Is Charging | Is-Charging signal status |
| 8   | `0x0100` | 256 | ✅ | MP Input 2 | Multi-Purpose Input #2 signal status |
| 9   | `0x0200` | 512 | ❌ | MP Input 3 | Multi-Purpose Input #3 signal status |
| 10  | `0x0400` | 1024 | ✅ | Reserved | RESERVED |
| 11  | `0x0800` | 2048 | ❌ | MP Output 2 | Multi-Purpose Output #2 signal status |
| 12  | `0x1000` | 4096 | ❌ | MP Output 3 | Multi-Purpose Output #3 signal status |
| 13  | `0x2000` | 8192 | ❌ | MP Output 4 | Multi-Purpose Output #4 signal status |
| 14  | `0x4000` | 16384 | ❌ | MP Enable | Multi-Purpose Enable signal status |
| 15  | `0x8000` | 32768 | ❌ | MP Output 1 | Multi-Purpose Output #1 signal status |

**Example Analysis**: `0x05E6` = 1,510 decimal → bits 1, 2, 5, 6, 7, 8, 10 active
- Charge relay enabled
- Charger safety asserted
- Always-on signal active
- System is ready
- Vehicle is charging
- Multi-Purpose Input #2 active
- Reserved bit 10 set (purpose unknown)

---

## Zero EV CSS Controller Messages

### From Orion BMS 2 to Controller
The following message IDs are sent from the BMS to the EV Controller:

- `0x351` - Control message 1
- `0x355` - Control message 2  
- `0x356` - Control message 3
- `0x357` - Control message 4
- `0x35A` - Control message 5
- `0x35B` - Control message 6

*Note: Detailed message structure for these IDs is not yet documented.*

---

## Message Frequency Analysis

### Sample Data from 2025-09-16

| CAN ID | Count | Rate | Description |
|--------|-------|------|-------------|
| `036` | 52 | Low | Battery Cell Broadcast |
| `076` | 262 | High | Thermistor Broadcast |
| `351` | 252 | High | BMS → Controller |
| `355` | 252 | High | BMS → Controller |
| `356` | 252 | High | BMS → Controller |
| `35A` | 52 | Low | BMS → Controller |
| `35B` | 52 | Low | BMS → Controller |
| `374-381` | 37-38 | Low | Unknown source |
| `6B0-6B4` | 26 | Low | Custom BMS frames |

### Message Categories
- **High Frequency** (250+ messages): Core control and monitoring
- **Low Frequency** (25-75 messages): Status updates and diagnostics
- **Very Low Frequency** (<40 messages): Diagnostic or configuration data

---

## Implementation Notes

1. **Voltage Precision**: Cell voltages use 4 decimal places (÷10000)
2. **Temperature Units**: Celsius, direct decimal conversion
3. **Current Limits**: May require additional scaling factors
4. **Checksums**: Algorithm not yet reverse-engineered
5. **Reserved Fields**: May contain diagnostic or future expansion data
