Kasa HomeKit Bridge
=========================

Control your TP-Link Kasa devices from Siri or the Home.app

Install
-------

`go install github.com/cloudkucooland/HomeKitBridges/KasaHKBridge@latest`

`sudo mkdir -p /var/db/HomeKitBridges/Kasa/`

`sudo chown -R `whoami` /var/db/HomeKitBridges`

Start the process

`~/go/bin/kasa-homekit`

Install into HomeKit
--------------------

On an iOS device, open the Home.app, add new accessory.

Click the "more options" link, the bridge should be visible in the list, click it.

Enter the default code 00102003

Firewall Considerations
-----------------------

If you are running a firewall on your system, you need to allow UDP and TCP on port 9999 inbound and outbound. Check your system's firewall documentation for details on how to do that.

Run via systemd on Linux distros that use systemd
-------------------------------------------------

As root, create a /etc/systemd/system/kasa.service file

```
[Unit]
Description=Kasa HomeKit Bridge
After=network-online.target

[Service]
User=scot
ExecStart=/home/scot/go/bin/kasa-homekit
Type=exec
Restart=on-failure
RestartSec=1
SuccessExitStatus=3 4
RestartForceExitStatus=3 4

[Install]
WantedBy=multi-user.target
Alias=kasa-homekit
```

`systemctl enable kasa`

`systemctl start kasa`
