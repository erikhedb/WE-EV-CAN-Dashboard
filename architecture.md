# System Architecture

This document describes the overall architecture of the EV information system, including the data flow and how the repository is organized.

---

## Overview

The system connects to the **CAN bus** of an EV (with an **Orion BMS 2** and **Tesla Model 3 RDU**) using a **Raspberry Pi 5 with a CAN interface**.  
The architecture is designed to:

- Collect raw CAN frames
- Parse and decode relevant values
- Continuously update JSON files with the latest data
- Provide a web-based interface for visualization

---

## Data Flow

```mermaid
flowchart LR
    A[CAN Bus\n(Orion BMS + Tesla RDU)] --> B[Logger\n(Python backend)]
    B -->|Writes JSON| C[Web Interface\n(Flask SPA)]
    B -->|Stores samples| D[Data/]
    E[Configs/] -->|Parameters| B
    F[Docs/] -.-> B
    F -.-> C