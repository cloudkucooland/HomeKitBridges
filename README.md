HomeKitBridges
==============

This repository contains several bridges to allow Apple HomeKit (i.e. Siri) to control devices which do not natively support HomeKit.

-	TP-Link Kasa switches
-	Daikin One Thermostat
-	Konnected.io Alarm systems

Requirements
------------

These bridges are written in Go. They are light-weight and fast. They run on my antique Linux server (2008 vintage) with no problem at all. I've tested them on Raspberry Pi Zero and they work fine there too. If your computer can run Go, these should run just fine.

Install
-------

See the directory for the individual bridge for full installation instructions.

The config files and HomeKit data is stored in `/var/db/HomeKitBridges` and the directory needs to be created before starting the processes.

`sudo mkdir -p /var/db/HomeKitBridges`

`sudo chown -R `whoami` /var/db/HomeKitBridges`

HomeKit PIN
-----------

The default HomeKit device pin is 00102003 .

The pin for the Konnected bridge is configured in its startup file.
