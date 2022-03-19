#!/bin/sh
#
# push a new config to the konnected.io device (not pro)
#
#{"actuators":[ {"pin":8, "trigger": 1} ]}
#
curl -X PUT -H "Content-Type: application/json" -d '{"endpoint_type":"rest","endpoint":"http://192.168.12.253:15301/konnected","token":"notyet","sensors":[{"pin":1},{"pin":2},{"pin":5},{"pin":6},{"pin":7}],"actuators":[{"pin":8},{"trigger":1}]}' http://192.168.12.176:15301/settings
