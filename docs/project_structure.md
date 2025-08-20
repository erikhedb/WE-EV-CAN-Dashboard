# Project Structure

## /sample/
Contains sample data frames from `candump` and Python tools to parse and understand the dump.  
Used for testing decoding logic without needing live CAN traffic.

    > log_stat.py file.can # Frame count in a can dump file
    # Frames total: 28764
    # By interface: can0=28764
    #
    Rank  CAN_ID       Count   Percent
    ------------------------------------
    1  6B0          19992    69.50%
    2  7EB           3848    13.38%
    3  7E3           2193     7.62%
    4  6B2           1571     5.46%
    5  6B1            828     2.88%
    6  1806E5F4       107     0.37%
    7  1806E7F4       107     0.37%
    8  1806E9F4       107     0.37%
    9  7DF             11     0.04%

    
## /logger/
Code for the Python-based backend logger.  
The logger continuously updates a set of JSON files with the latest values from the CAN bus.  

## /web/
Code for the user interface.  
This is a single-page Python3 Flask application that displays the JSON content in a structured dashboard.  

## /docs/
Documentation files such as `hardware.md`, `architecture.md`, and other design notes.  
Provides reference material and project planning information.  

## /config/
Configuration files for CAN interfaces, logging parameters, and system setup.  
Keeps hardware- and environment-specific settings separate from code. 