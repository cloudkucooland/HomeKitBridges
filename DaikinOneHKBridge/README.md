Daikin One HomeKit Bridge
=========================

Control your Daikin One Thermostat from Siri or the Home.app

Install
-------

`go install github.com/cloudkucooland/HomeKitBridges/DaikinOneHKBridge@latest`

`mv ~/go/main ~/go/daikin-homekit` (because I need to fix this...)

`sudo mkdir -p /var/db/HomeKitBridges/Daikin/`

`sudo chown -R \`whoami\` /var/db/HomeKitBridges\`

Edit /var/db/HomeKitBridges/Daikin/daikin.json file to contain your API info (vi, nano, whatever editor you use)

```
{
    "Email": "you@example.com",
    "Password": "letmein"
}
```

Start the process

`~/go/bin/daikin-homekit`

Install into HomeKit
--------------------

On an iOS device, open the Home.app, add new accessory.

Click the "more options" link, the bridge should be visible in the list, click it.

Enter the default code 00102003

Run via systemd on Linux distros that use systemd
-------------------------------------------------

As root, create a /etc/systemd/system/daikin.service file

```
[Unit]
Description=Daikin HomeKit Bridge
After=network-online.target

[Service]
User=scot
ExecStart=/home/scot/go/bin/daikin-homekit
Type=exec
Restart=on-failure
RestartSec=1
SuccessExitStatus=3 4
RestartForceExitStatus=3 4

[Install]
WantedBy=multi-user.target
Alias=daikin-homekit
```

`systemctl enable daikin`

`systemctl start daikin`
