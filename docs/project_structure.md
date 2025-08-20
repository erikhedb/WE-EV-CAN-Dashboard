# Project Structure

## /sample/
Contains sample data frames from `candump` and Python tools to parse and understand the dump.  
Used for testing decoding logic without needing live CAN traffic.  

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