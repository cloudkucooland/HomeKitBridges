Daikin One HomeKit Bridge
=========================

Control your Konnected.io devices from Siri or the Home.app

Install
-------

`go install github.com/cloudkucooland/HomeKitBridges/KonnectedHKBridge@latest`

`sudo mkdir -p /var/db/HomeKitBridges/Konnected/`

`sudo chown -R`whoami`/var/db/HomeKitBridges`

Configure the /var/db/HomeKitBridges/Konnected/khkb.json file to match your config

```
{
    "Pin":"00102003",
    "ListenAddr":"{server IP:port}",
    "Devices": [ 
     {
      "Mac": "f4cfa26a0000",
      "Password": "password",
      "Zones":[
        {"pin":1, "name":"Front Door", "type":"door"},
        {"pin":2, "name":"Garage Door", "type":"door"},
        {"pin":5, "name":"Back Door", "type":"door"},
        {"pin":6, "name":"Attic Hatch", "type":"door"},
        {"pin":7, "name":"Motion Sensor", "type":"motion"},
        {"pin":8, "name":"Buzzer", "type":"buzzer"},
        {"pin":9, "name":"Unused", "type":"unused"}
      ]
      }
   ]
}
```

Pin is the HomeKit pin to use. The default is 00102003. This is what you use when adding it via Home.app on iOS.

ListenAddr is the address and port of the bridge. Make sure you use a valid IP address of your server and the port (TCP) is open on your server's firewall. I use port 8889 for no good reason.

You should be able to add multiple devices, but I've not tested this.

A device's Mac is it's hardware address. Get this from the device via the Konnected.app.

The Password is the token to be set during provisioning (see below).

Start the process

`~/go/bin/khkb`

At this point it will attempt to discover a device. For now you must manually provision the device using CURL. The server process must be running with the Mac and Password/token of the device configured. (see historical note below as to why it doesn't yet auto-provision)

(note to self, change Password to Token everywhere, just to keep it consistent)

curl -X PUT -H "Content-Type: application/json" -d '{"endpoint_type":"rest","endpoint":"http://{server IP:port}/konnected","token":"password", "sensors":[{"pin":1},{"pin":2},{"pin":5},{"pin":6},{"pin":7}] }' http://{konnected ip:port from Konnected.app}/settings

Install into HomeKit
--------------------

On an iOS device, open the Home.app, add new accessory.

Click the "more options" link, the bridge should be visible in the list, click it.

Enter the default code configured in the khkb.json file (default is 00102003)

Firewall Considerations
-----------------------

If you are running a firewall on your system, you need to allow TCP on port 8889 (or whatever you configured in the khkb.json file) inbound and outbound. Check your system's firewall documentation for details on how to do that.

Run via systemd on Linux distros that use systemd
-------------------------------------------------

As root, create a /etc/systemd/system/konnected.service file

```
[Unit]
Description=Konnected HomeKit Bridge
After=network-online.target

[Service]
User=scot
ExecStart=/home/scot/go/bin/khkb
Type=exec
Restart=on-failure
RestartSec=1
SuccessExitStatus=3 4
RestartForceExitStatus=3 4

[Install]
WantedBy=multi-user.target
Alias=konnected-homekit
```

`systemctl enable konnected`

`systemctl start konnected`

Historical Note
---------------

The first Konnected unit I got was glitchy. I had a lot of trouble developing the startup/setup logic. I thought I was doing things wrong. It turned out to be faulty hardware. Konnected kindly replaced that first unit and the second unit has been rock solid. I've not gone back and added the auto-provision feature that I had intended. This install/setup should be easier than it is. I'm just reluctant to make change because what I've got is working for me.
