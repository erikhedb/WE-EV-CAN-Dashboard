# Information on running a CANBUS-shield on Rasberry Pi 5

[Link to more information about the 2-SH CAN HAT](https://www.waveshare.com/wiki/2-CH_CAN_HAT)


Basic SPI configuration can be found in /boot/firmware/config.txt

    # Enable SPI1 with 3 chip-selects
    dtoverlay=spi1-3cs

    # MCP2515 on SPI1.2  ->  can0  ->  INT = GPIO13
    dtoverlay=mcp2515,spi1-2,oscillator=16000000,interrupt=13

    # MCP2515 on SPI1.1  ->  can1  ->  INT = GPIO22
    dtoverlay=mcp2515,spi1-1,oscillator=16000000,interrupt=22


## Commands

Show can0 details

    ip -details link show can0

Start can0 in listen mode

    sudo ip link set can0 type can bitrate 500000 listen-only on


Bring it online 
    
    sudo ip link set can0 up


Dump messages to a log file
    
    candump -t a can0 > can0.log &
