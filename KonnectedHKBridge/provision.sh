#!/bin/sh
#
#{"actuators":[ {"pin":8, "trigger": 1} ]}
#
curl --request POST --header "content-type: application/json" --data '{"endpoint_type":"rest","endpoint":"http://192.168.100.5:8999/konnected","token":"notyet", "sensors":[{"pin":1},{"pin":2},{"pin":5},{"pin":6},{"pin":7}] }' http://192.168.100.63:14996/settings
