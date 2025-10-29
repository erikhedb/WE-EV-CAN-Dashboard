# CAN Bus Message Documentation

## Overview

This document describes the CAN bus messages used in the EV system, primarily focusing on communication between the Orion BMS 2 and the Zero EV CSS Controller.

## Data Format Specifications

- **Bit Order**: Most significant bit first (MSB) → `00000001b = 1` decimal
- **Byte Order**: Big Endian (Most significant byte first) → `0001h = 1` decimal

---

## Message Metadata

| CAN ID | Name | Source | Description |
|--------|------|--------|-------------|
| `0x036` | Battery Cell Broadcast | Orion BMS 2 | Aggregate cell voltage and balancing status payload broadcast on the standard CAN ID. |
| `0x076` | Thermistor Broadcast | Orion BMS 2 | Standard thermistor frame with the latest pack temperature readings used by the charger and ECU. |
| `0x125` | DU1Feedback | Drive Unit | Real-time inverter telemetry: throttle torque request, DC bus current, AC current, and bus voltage. |
| `0x126` | DU1Status | Drive Unit | Inverter operating state flags plus motor/inverter temperatures, motor speed, gear selection, and operating mode. |
| `0x127` | DU1Diagnostic | Drive Unit | Diagnostic bit-field indicating whether torque, current, voltage, or temperature limits are constraining the drive unit. |
| `0x351` | BmsLimits | Orion BMS 2 | Charge/discharge current and voltage limits that the controller should respect for the HV battery. |
| `0x355` | BmsSOC | Orion BMS 2 | State-of-charge metrics: pack SOC, high-definition SOC, and state-of-health. |
| `0x356` | BmsStatus1 | Orion BMS 2 | Pack voltage, pack current, and aggregate HV battery temperature. |
| `0x357` | BMSCCSCommands | Orion BMS 2 | CCS interface commands including AC current limit requests and isolation relay override flag. |
| `0x35A` | BmsErrors | Orion BMS 2 | Bit-coded diagnostic trouble codes spanning pack isolation, thermistors, current sensors, and relay faults. |
| `0x35B` | BmsStatus2 | Orion BMS 2 | Relay/contactor status bits along with the isolation monitor reading. |
| `0x6B0` | Battery Pack Status | Orion BMS 2 | Custom frame with State of Charge, cell count, pack voltage, and checksum. |
| `0x6B1` | High Cell Information | Orion BMS 2 | Reports highest cell ID, instantaneous pack current (signed), and high cell voltage. |
| `0x6B2` | Low Cell Information | Orion BMS 2 | Reports lowest cell ID, 12 V auxiliary rail voltage, and low cell voltage. |
| `0x6B3` | Temperature Information | Orion BMS 2 | Reports highest and lowest temperature sensor readings. |
| `0x6B4` | System Control Information | Orion BMS 2 | Contains relay state bit-field plus charge/discharge current limits. |

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

## Zero EV CSS Controller Interface (DBC-Derived)

The Zero EV DBC maps the Orion BMS broadcast frames that the EV controller consumes:

- `0x351` (`BmsLimits`): Charge/discharge current and voltage ceilings that the inverter should honor.
- `0x355` (`BmsSOC`): Pack state of charge (coarse and high-definition) plus state of health.
- `0x356` (`BmsStatus1`): Instantaneous pack voltage, pack current, and weighted HV battery temperature.
- `0x357` (`BMSCCSCommands`): CCS/AC charging commands including AC current limit request and isolation relay override.
- `0x35A` (`BmsErrors`): Bit-coded DTC summary covering isolation, sensor, relay, and thermal faults.
- `0x35B` (`BmsStatus2`): Relay/contactor status bits (MP outputs, charge/discharge relays) and isolation monitor reading.

These IDs align with the higher-frequency messages observed in log captures and replace the placeholder “control frame” labels used previously.

### Drive Unit Messages
- `0x125` (`DU1Feedback`): Drive Unit feedback frame with torque request, DC bus current, AC phase current, and bus voltage.
- `0x126` (`DU1Status`): Operating state, inverter/motor temperatures, motor speed, gear selection, and drive mode flags.
- `0x127` (`DU1Diagnostic`): Diagnostic limit flags indicating whether torque, current, voltage, or temperature limits are constraining the drive.

---

## Message Frequency Analysis

### Sample Data from 2025-09-16

| CAN ID | Count | Rate | Description |
|--------|-------|------|-------------|
| `036` | 52 | Low | Battery Cell Broadcast |
| `076` | 262 | High | Thermistor Broadcast |
| `351` | 252 | High | BmsLimits |
| `355` | 252 | High | BmsSOC |
| `356` | 252 | High | BmsStatus1 |
| `35A` | 52 | Low | BmsErrors |
| `35B` | 52 | Low | BmsStatus2 |
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
